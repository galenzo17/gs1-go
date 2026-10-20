// gs1-loader.js — Load the GS1 WASM module.
// Requires wasm_exec.js to be loaded first (provides the Go class).

/**
 * Load and initialize the GS1 WASM module.
 * @param {string} wasmPath - Path or URL to the gs1.wasm file.
 * @returns {Promise<typeof gs1>} The gs1 namespace with parse, parseEPC, encode, validateGTIN, validateKey, validateRegulatory, newSSCC, newGTIN13, newGTIN14.
 */
export async function loadGS1(wasmPath) {
  if (typeof Go === "undefined") {
    throw new Error(
      "wasm_exec.js must be loaded before gs1-loader.js (provides the Go class)"
    );
  }

  const go = new Go();

  let result;
  if (typeof WebAssembly.instantiateStreaming === "function") {
    result = await WebAssembly.instantiateStreaming(
      fetch(wasmPath),
      go.importObject
    );
  } else {
    // Safari <15 fallback: fetch + instantiate separately.
    const resp = await fetch(wasmPath);
    const bytes = await resp.arrayBuffer();
    result = await WebAssembly.instantiate(bytes, go.importObject);
  }

  // Run the Go main — this registers gs1.* on globalThis.
  go.run(result.instance);

  return globalThis.gs1;
}
