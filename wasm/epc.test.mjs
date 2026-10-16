import assert from "node:assert/strict";
import { webcrypto } from "node:crypto";
import { readFile } from "node:fs/promises";
import { setTimeout as delay } from "node:timers/promises";

// Run after make wasm; wasm_exec.js must match the build's Go toolchain.
const processTimeout = setTimeout(() => {
  console.error("WASM EPC tests exceeded the 15-second process timeout");
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

  const sgtin = gs1.parseEPC("3074257BF7194E4000001A85");
  assert.equal(sgtin.symbology, "EPC");
  assert.equal(sgtin.gtin, "80614141123458");
  assert.equal(sgtin.serial, "6789");
  assert.deepEqual(sgtin.elements, [{ ai: "01", value: "80614141123458" }, { ai: "21", value: "6789" }]);
  assert.deepEqual(sgtin.epc, {
    scheme: "sgtin", bits: 96, filter: 3, companyPrefix: "0614141", reference: "812345", serial: "6789",
    uri: "urn:epc:id:sgtin:0614141.812345.6789",
  });

  const sscc = gs1.parseEPC("urn:epc:tag:sscc-96:3.0614141.1234567890");
  assert.deepEqual(sscc.elements, [{ ai: "00", value: "106141412345678908" }]);
  assert.equal(sscc.epc.bits, 96);
  assert.equal(sscc.epc.filter, 3);

  const sgln = gs1.parseEPC("urn:epc:id:sgln:0614141.12345.400");
  assert.equal(sgln.gln, "0614141123452");
  assert.deepEqual(sgln.elements, [{ ai: "414", value: "0614141123452" }, { ai: "254", value: "400" }]);
  assert.ok(!("bits" in sgln.epc) && !("filter" in sgln.epc));

  assert.match(gs1.parseEPC("350000000000000000000000").error, /unsupported EPC scheme/);
  assert.match(gs1.parseEPC("3G74").error, /non-hexadecimal/);
  assert.match(gs1.parseEPC("urn:epc:id:gdti:0614141.12345.ABC").error, /unsupported EPC scheme/);
  assert.match(gs1.parseEPC().error, /requires 1 argument/);

  await delay(0);
  if (runtimeError) throw runtimeError;
  console.log("WASM EPC tests passed");
  clearTimeout(processTimeout);
  process.exit(0);
} catch (error) {
  console.error(error);
  clearTimeout(processTimeout);
  process.exit(1);
}
