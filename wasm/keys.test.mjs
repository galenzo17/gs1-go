import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import { readFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

// Run after make wasm; wasm_exec.js must match the build's Go toolchain.
const processTimeout = setTimeout(() => {
  console.error("WASM key tests exceeded the 15-second process timeout");
  process.exit(1);
}, 15_000);

try {
  globalThis.crypto ??= webcrypto;
  await import(new URL("./wasm_exec.js", import.meta.url).href);
  const go = new globalThis.Go();
  const bytes = await readFile(new URL("./gs1.wasm", import.meta.url));
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  let runtimeError;
  go.run(instance).then(
    () => { runtimeError = new Error("Go WASM runtime exited unexpectedly"); },
    (error) => { runtimeError = error; },
  );
  const startupDeadline = Date.now() + 5_000;
  while (!globalThis.gs1) {
    if (runtimeError) throw runtimeError;
    assert.ok(Date.now() < startupDeadline, "WASM parser did not start within 5 seconds");
    await delay(5);
  }
  const { gs1 } = globalThis;

  // validateKey reports every family a length admits.
  assert.deepEqual(gs1.validateKey("106141411234567897"), { valid: true, keyTypes: ["SSCC", "GSRN"] });
  assert.deepEqual(gs1.validateKey("0614141000012"), { valid: true, keyTypes: ["GTIN", "GLN"] });
  assert.deepEqual(gs1.validateKey("06141410000000012"), { valid: true, keyTypes: ["GSIN"] });
  assert.deepEqual(gs1.validateKey("04150000021126"), { valid: true, keyTypes: ["GTIN"] });

  const badCheck = gs1.validateKey("106141411234567891");
  assert.equal(badCheck.valid, false);
  assert.deepEqual(badCheck.keyTypes, []);
  assert.match(badCheck.error, /invalid check digit/);

  const badLength = gs1.validateKey("123456789");
  assert.equal(badLength.valid, false);
  assert.match(badLength.error, /matches no GS1 key/);
  assert.match(gs1.validateKey().error, /requires 1 argument/);

  // validateCheckDigits covers SSCC, GLN, GSIN, GSRN, GDTI and GRAI.
  const keys = [
    { valid: "00106141411234567897", invalid: "00106141411234567891", ai: "00" },
    { valid: "4140614141000012", invalid: "4140614141000013", ai: "414" },
    { valid: "40206141410000000012", invalid: "40206141410000000013", ai: "402" },
    { valid: "8017061414112345678902", invalid: "8017061414112345678903", ai: "8017" },
    { valid: "2530614141000012ABC", invalid: "2530614141000013ABC", ai: "253" },
    { valid: "800300614141000012XYZ", invalid: "800300614141000013XYZ", ai: "8003" },
  ];
  for (const key of keys) {
    assert.ok(!("error" in gs1.parse(key.valid)), `lenient ${key.ai}`);
    assert.ok(!("error" in gs1.parse(key.invalid)), `lenient invalid ${key.ai}`);
    assert.ok(!("error" in gs1.parse(key.valid, { validateCheckDigits: true })), `checked ${key.ai}`);
    const result = gs1.parse(key.invalid, { validateCheckDigits: true });
    assert.deepEqual(Object.keys(result), ["error"], `checked invalid ${key.ai}`);
    assert.match(result.error, /invalid check digit/);
    assert.match(result.error, new RegExp(`AI \\(${key.ai}\\)`));
  }

  await delay(0);
  if (runtimeError) throw runtimeError;
  console.log("WASM key tests passed");
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
