import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import { readFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

// Run after make wasm; wasm_exec.js must match the build's Go toolchain.
// Checks the WASM result shapes that mirror the CLI JSON output.
const processTimeout = setTimeout(() => {
  console.error("WASM parity tests exceeded the 15-second process timeout");
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

  // Element names and measures mirror the CLI JSON output.
  const weighed = gs1.parse("0104150000021126" + "3102001500" + "3405001250");
  assert.deepEqual(weighed.elements, [
    { ai: "01", name: "GTIN", value: "04150000021126" },
    { ai: "3102", name: "Net Weight kg", value: "001500", measure: { raw: "001500", value: 15, scaled: 1500, decimals: 2, unit: "kg" } },
    { ai: "3405", name: "Gross Weight lb", value: "001250", measure: { raw: "001250", value: 0.0125, scaled: 1250, decimals: 5, unit: "lb" } },
  ]);
  const dated = gs1.parse("0104150000021126" + "17250200" + "12250630", { dateFormat: "iso", dayZero: "first" }).elements;
  assert.equal(dated[1].date, "2025-02-01");
  assert.equal(dated[2].date, "2025-06-30");
  assert.ok(!("date" in gs1.parse("17250630").elements[0]));
  const internal = gs1.parse("90ABC").elements[0];
  assert.equal(internal.name, "Internal");
  assert.ok(!("measure" in internal));

  // assumeBareGTIN8 mirrors ParseOptions.AssumeBareGTIN8 and the CLI -gtin8 flag.
  assert.deepEqual(gs1.parse("96385074").elements.map((e) => e.ai), ["96"]);
  assert.equal(gs1.parse("96385074", { assumeBareGTIN8: true }).gtin, "00000096385074");
  assert.match(gs1.parse("96385075", { assumeBareGTIN8: true, validateCheckDigits: true }).error, /invalid check digit/);
  assert.equal(gs1.parse("96385074", { assumeBareGTIN8: "true" }).gtin, "");

  // Builders return the key or { error }.
  assert.equal(gs1.newSSCC("3", "7801234", "000000009"), "378012340000000095");
  assert.match(gs1.newSSCC("3", "7", "000000009").error, /company prefix/);
  assert.match(gs1.newSSCC("33", "7801234", "000000009").error, /single digit/);
  assert.match(gs1.newSSCC("3").error, /requires 3 arguments/);
  assert.match(gs1.newSSCC(3, "7801234", "000000009").error, /extensionDigit must be a string/);
  assert.match(gs1.newGTIN14("1", gs1.newGTIN13("7801234", "5")).error, /gtin13 must be a string/);
  assert.equal(gs1.newGTIN13("7801234", "00005"), "7801234000056");
  assert.match(gs1.newGTIN13("7801234", "5").error, /12 digits/);
  assert.match(gs1.newGTIN13("7801234").error, /requires 2 arguments/);
  assert.equal(gs1.newGTIN14("1", "7801234000056"), "17801234000053");
  assert.match(gs1.newGTIN14("1", "7801234000057").error, /invalid check digit/);
  assert.match(gs1.newGTIN14("12", "7801234000056").error, /single digit/);
  assert.match(gs1.newGTIN14("1").error, /requires 2 arguments/);
  assert.ok(gs1.validateGTIN(gs1.newGTIN14("1", gs1.newGTIN13("7801234", "00005"))));

  await delay(0);
  if (runtimeError) throw runtimeError;
  console.log("WASM parity tests passed");
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
