import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import { readFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

// Run after make wasm; wasm_exec.js must match the build's Go toolchain.
const processTimeout = setTimeout(() => {
  console.error("WASM encode tests exceeded the 15-second process timeout");
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
  const GS = "\x1D";

  // Array input, predefined-length AIs first, FNC1 only where required.
  const logistic = [
    { ai: "10", value: "ABC123" },
    { ai: "00", value: "790123451234500116" },
    { ai: "02", value: "49012345008794" },
    { ai: "37", value: "12" },
  ];
  assert.deepEqual(gs1.encode(logistic), {
    raw: `${GS}00790123451234500116024901234500879410ABC123${GS}3712`,
    hri: "(00)790123451234500116(02)49012345008794(10)ABC123(37)12",
    zpl: ">800790123451234500116024901234500879410ABC123>83712",
  });

  // Bracket notation input and a raw scan string give the same result.
  const fromBrackets = gs1.encode("(10)LOT42(01)04150000021126(21)SERIAL");
  assert.deepEqual(fromBrackets, {
    raw: `${GS}010415000002112610LOT42${GS}21SERIAL`,
    hri: "(01)04150000021126(10)LOT42(21)SERIAL",
    zpl: ">8010415000002112610LOT42>821SERIAL",
  });
  assert.deepEqual(gs1.encode(fromBrackets.raw), fromBrackets);
  assert.deepEqual(gs1.encode(fromBrackets.hri), fromBrackets);

  // Literal > is escaped for ZPL only.
  const gt = gs1.encode([{ ai: "01", value: "04150000021126" }, { ai: "10", value: "A>B" }]);
  assert.equal(gt.raw, `${GS}010415000002112610A>B`);
  assert.equal(gt.zpl, ">8010415000002112610A><B");

  // The raw output parses under both strict options.
  const parsed = gs1.parse(fromBrackets.raw, { strict: true, validateCheckDigits: true });
  assert.deepEqual(parsed.elements.map(({ ai, value }) => ({ ai, value })), [
    { ai: "01", value: "04150000021126" }, { ai: "10", value: "LOT42" }, { ai: "21", value: "SERIAL" },
  ]);

  // Errors: associations, check digits, unknown AI, malformed input.
  assert.match(gs1.encode("(02)49012345008794(37)12").error, /AI \(37\) requires AI \(00\)/);
  assert.match(gs1.encode([{ ai: "21", value: "S" }]).error, /AI \(21\) requires/);
  assert.match(gs1.encode([{ ai: "01", value: "04150000021127" }]).error, /invalid check digit/);
  assert.match(gs1.encode([{ ai: "999", value: "1" }]).error, /unknown application identifier/);
  assert.match(gs1.encode("(999)1").error, /unknown application identifier/);
  assert.match(gs1.encode("(01").error, /invalid data/);
  assert.match(gs1.encode([{ ai: "10", value: "A" }, { ai: "10", value: "B" }, { ai: "01", value: "04150000021126" }]).error, /duplicate AI/);
  assert.match(gs1.encode([]).error, /empty input/);
  assert.match(gs1.encode().error, /requires 1 argument/);
  assert.match(gs1.encode(42).error, /expects an array/);
  assert.match(gs1.encode({ ai: "01" }).error, /expects an array/);
  assert.match(gs1.encode(["01"]).error, /element 0 is not an object/);
  assert.match(gs1.encode([{ ai: 1, value: "x" }]).error, /element 0 needs string ai and value/);

  await delay(0);
  if (runtimeError) throw runtimeError;
  console.log("WASM encode tests passed");
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
