import assert from "node:assert/strict";
import { setTimeout as delay } from "node:timers/promises";

// Smoke test of the npm package through its own loader: run after make wasm
// (and cp LICENSE wasm/ when checking the package contents). It must import
// wasm_exec.js and gs1.wasm relative to gs1-loader.js without arguments.
const processTimeout = setTimeout(() => {
  console.error("npm package smoke test exceeded the 15-second process timeout");
  process.exit(1);
}, 15_000);

try {
  const { loadGS1 } = await import("./gs1-loader.js");
  const gs1 = await loadGS1();
  const startupDeadline = Date.now() + 5_000;
  while (!globalThis.gs1) {
    assert.ok(Date.now() < startupDeadline, "WASM parser did not start within 5 seconds");
    await delay(5);
  }
  assert.equal(gs1, globalThis.gs1);

  const r = gs1.parse("0104150000021126172506301012345", { dateFormat: "iso" });
  assert.equal(r.gtin, "04150000021126");
  assert.equal(r.expirationDate, "2025-06-30");
  assert.equal(r.lot, "12345");
  assert.equal(gs1.validateGTIN("04150000021126"), true);
  assert.equal(gs1.validateRegulatory("0104150000021126172506301012345", "cofepris"), null);

  // An explicit URL keeps working for custom hosting.
  const again = await loadGS1(new URL("./gs1.wasm", import.meta.url));
  assert.equal(typeof again.parse, "function");

  const pkg = (await import("./package.json", { with: { type: "json" } })).default;
  assert.equal(pkg.name, "@galenzo17/gs1-wasm");
  assert.equal(pkg.exports["."].import, "./gs1-loader.js");
  assert.equal(pkg.exports["."].types, "./gs1.d.ts");

  console.log(`npm package smoke test passed (${pkg.name}@${pkg.version})`);
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
