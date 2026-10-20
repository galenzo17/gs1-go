# @galenzo17/gs1-wasm

GS1 Application Identifier parser and validator for browsers, Electron and
Node, compiled to WebAssembly from [gs1-go](https://github.com/galenzo17/gs1-go).
It parses the element strings carried by GS1-128, GS1 DataMatrix, GS1 QR
Code, EAN/UPC and ITF-14 symbols, validates check digits and association
rules, and knows the healthcare regulators of Latin America. No dependencies.

```bash
npm install @galenzo17/gs1-wasm
```

## Usage

With a bundler (Vite, webpack, esbuild) or in Node 20 and later:

```js
import { loadGS1 } from "@galenzo17/gs1-wasm";

const gs1 = await loadGS1();
const r = gs1.parse("0104150000021126172506301012345", { dateFormat: "iso" });
r.gtin;            // "04150000021126"
r.expirationDate;  // "2025-06-30"
r.lot;             // "12345"
gs1.validateGTIN("04150000021126");         // true
gs1.validateRegulatory("0104150000021126172506301012345", "anvisa"); // null when compliant, else a message
```

From a CDN without a build step:

```html
<script type="module">
  import { loadGS1 } from "https://cdn.jsdelivr.net/npm/@galenzo17/gs1-wasm@0.2.0/gs1-loader.js";
  const gs1 = await loadGS1();
  console.log(gs1.parse("(01)04150000021126(10)LOT42"));
</script>
```

The loader finds `gs1.wasm` and `wasm_exec.js` next to itself through
`import.meta.url`, so no path configuration is needed.
With Vite, exclude the package from dependency pre-bundling so the two files
resolve from the package directory instead of `node_modules/.vite/deps`:

```js
// vite.config.js
export default { optimizeDeps: { exclude: ["@galenzo17/gs1-wasm"] } };
```
 To host the binary
elsewhere, pass its URL: `loadGS1(new URL("/static/gs1.wasm", location))`.

TypeScript declarations ship in `gs1.d.ts` and resolve automatically;
`GS1ParseResult`, `GS1ParseOptions` and the other types are exported.

## Versions

The package version equals the gs1-go git tag it was built from, and every
release carries npm provenance linking it to the GitHub Actions run. The
WebAssembly binary is the single implementation; there is no JavaScript
rewrite. Issues and source live in the gs1-go repository.
