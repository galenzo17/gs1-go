// Fails unless npm pack would ship exactly the published files. Run from the
// wasm directory after make wasm and cp LICENSE wasm/:
//   npm pack --dry-run --json | node pack-check.mjs
import { readFileSync } from "node:fs";

const [pkg] = JSON.parse(readFileSync(0, "utf8"));
const files = pkg.files.map((f) => f.path).sort();
const want = ["LICENSE", "README.md", "gs1-loader.js", "gs1.d.ts", "gs1.wasm", "package.json", "wasm_exec.js"];
if (JSON.stringify(files) !== JSON.stringify(want)) {
  console.error("npm pack would ship", files, "want", want);
  process.exit(1);
}
console.log("npm pack ships", files.join(", "));
