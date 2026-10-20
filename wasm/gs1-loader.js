// gs1-loader.js — Load the GS1 WASM module in browsers, Electron and Node.
//
// The module resolves gs1.wasm and wasm_exec.js next to this file through
// import.meta.url, so bundlers, CDNs (jsDelivr, unpkg) and Node resolve them
// without a path argument. Pass an explicit URL to host the binary elsewhere.

/**
 * Load and initialize the GS1 WASM module.
 * @param {string | URL} [wasmPath] - URL or path of gs1.wasm. Defaults to the
 *   gs1.wasm shipped next to this loader.
 * @returns {Promise<typeof gs1>} The gs1 namespace: parse, validateGTIN,
 *   validateRegulatory and the other entry points declared in gs1.d.ts.
 */
export async function loadGS1(wasmPath) {
  if (typeof globalThis.Go === "undefined") {
    // wasm_exec.js from the Go toolchain defines globalThis.Go. It is a
    // classic script, so importing it as a module only runs it for effect.
    await import(new URL("./wasm_exec.js", import.meta.url).href);
  }
  // The default is a literal new URL(..., import.meta.url) so bundlers such
  // as Vite add gs1.wasm to the production asset graph.
  let url;
  if (wasmPath === undefined) {
    url = new URL("./gs1.wasm", import.meta.url);
  } else if (wasmPath instanceof URL) {
    url = wasmPath;
  } else {
    url = new URL(String(wasmPath), import.meta.url);
  }

  const go = new globalThis.Go();
  const instance = await instantiate(url, go.importObject);

  // Run the Go main — this registers gs1.* on globalThis and keeps the
  // runtime alive; the promise settles only if the runtime exits.
  go.run(instance);

  return globalThis.gs1;
}

async function instantiate(url, importObject) {
  if (url.protocol === "file:") {
    // Node and Electron main process: fetch does not serve file URLs.
    const { readFile } = await import("node:fs/promises");
    const bytes = await readFile(url);
    return (await WebAssembly.instantiate(bytes, importObject)).instance;
  }
  if (typeof WebAssembly.instantiateStreaming === "function") {
    try {
      return (await WebAssembly.instantiateStreaming(fetch(url), importObject)).instance;
    } catch {
      // Servers that send the wrong MIME type fall through to the buffered path.
    }
  }
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`gs1-wasm: fetching ${url} failed with HTTP ${response.status}`);
  }
  const bytes = await response.arrayBuffer();
  return (await WebAssembly.instantiate(bytes, importObject)).instance;
}
