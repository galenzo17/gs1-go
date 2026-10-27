// Package symbol renders GS1 keys as barcode symbols.
//
// The retail and logistic linear symbologies take a GS1 key. EAN13, EAN8,
// UPCA and UPCE encode a GTIN as the EAN/UPC symbols of ISO/IEC 15420 and
// ITF14 encodes a GTIN-14 as the ITF-14 symbol of ISO/IEC 16390. Each returns
// a Linear that carries the bar pattern together with the layout the GS1
// General Specifications prescribe around it: bar height, guard bar
// extension, bearer bars, quiet zones and the human readable interpretation
// (HRI). Render it with Image, PNG or SVG, choosing the pixels per module to
// match the X-dimension of the target printer.
//
// Dimensions are expressed in modules (X) at the nominal size of each
// symbology, so a scale of n pixels per module reproduces the GS1 nominal
// geometry when the printer resolution matches. Converting an X-dimension in
// millimeters and a printer resolution into a scale is left to the caller.
//
// Like the parent module, the package has no dependencies outside the
// standard library.
package symbol
