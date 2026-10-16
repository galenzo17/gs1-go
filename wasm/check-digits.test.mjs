import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import { readFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

// Run after make wasm; wasm_exec.js must match the build's Go toolchain.
const processTimeout = setTimeout(() => {
  console.error("WASM check-digit tests exceeded the 15-second process timeout");
  process.exit(1);
}, 15_000);

try {
  globalThis.crypto ??= webcrypto;
  await import(new URL("./wasm_exec.js", import.meta.url).href);
  const go = new globalThis.Go();
  const bytes = await readFile(new URL("./gs1.wasm", import.meta.url));
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  let runtimeError;
  // The production main stays alive indefinitely; observe failure without
  // awaiting its permanent run loop.
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

  const options = [
    {},
    { strict: true },
    { validateCheckDigits: true },
    { strict: true, validateCheckDigits: true },
  ];
  const inputs = [
    { input: "0104150000021126" },
    { input: "0001234567890123456702041500000211263720" },
    { input: "0104150000021127", checksum: true },
    { input: "0001234567890123456702041500000211273720", checksum: true },
    { input: "0204150000021126", association: true },
    { input: "0204150000021127", checksum: true, association: true },
  ];
  for (const opts of options) {
    for (const fixture of inputs) {
      const result = globalThis.gs1.parse(fixture.input, opts);
      const context = JSON.stringify({ fixture, opts });
      if (opts.strict && fixture.association) {
        assert.deepEqual(Object.keys(result), ["error"], context);
        assert.match(result.error, /invalid AI association/, context);
      } else if (opts.validateCheckDigits && fixture.checksum) {
        assert.deepEqual(Object.keys(result), ["error"], context);
        assert.match(result.error, /invalid check digit/, context);
      } else {
        assert.ok(!("error" in result), context);
        assert.equal(result.raw, fixture.input, context);
        assert.ok(result.elements.length > 0, context);
      }
    }
  }

  const carriers = [
    { valid: "5901234123457", invalid: "5901234123458", gtin: "05901234123457" },
    { valid: "]E496385074", invalid: "]E496385075", gtin: "00000096385074" },
    { valid: "]E0036000291452", invalid: "]E0036000291453", gtin: "00036000291452" },
    { valid: "]E000123457", invalid: "]E000123458", gtin: "00001234000057" },
    { valid: "]I012345678901231", invalid: "]I012345678901232", gtin: "12345678901231" },
  ];
  for (const fixture of carriers) {
    assert.equal(globalThis.gs1.parse(fixture.valid, { validateCheckDigits: true }).gtin, fixture.gtin);
    assert.ok(!("error" in globalThis.gs1.parse(fixture.invalid, { strict: true })));
    const result = globalThis.gs1.parse(fixture.invalid, { validateCheckDigits: true });
    assert.deepEqual(Object.keys(result), ["error"]);
    assert.match(result.error, /invalid check digit/);
  }
  assert.equal(globalThis.gs1.parse("]I20123456789012", { validateCheckDigits: true }).gtin, "01234567890128");
  assert.deepEqual(globalThis.gs1.parse("123456789013", { validateCheckDigits: true }).elements, [
    { ai: "12", name: "Due Date", value: "345678" }, { ai: "90", name: "Internal", value: "13" },
  ]);
  assert.ok(!("error" in globalThis.gs1.parse("0104150000021127", { validateCheckDigits: false })));
  assert.ok(!("error" in globalThis.gs1.parse("0104150000021127", { validateCheckDigits: "true" })));
  assert.match(globalThis.gs1.parse("0104150000021127XX", options[3]).error, /unknown application identifier/);
  const positioned = globalThis.gs1.parse("]d2(17)250630(01)04150000021127", { validateCheckDigits: true });
  assert.match(positioned.error, /position 12/);
  assert.match(positioned.error, /AI \(01\)/);

  await delay(0);
  if (runtimeError) throw runtimeError;
  console.log("WASM check-digit tests passed");
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
