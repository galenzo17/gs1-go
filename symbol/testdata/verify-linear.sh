#!/usr/bin/env bash
# Renders EAN-13, EAN-8, UPC-A, UPC-E and ITF-14 symbols at several scales
# and decodes the PNG files with an independent decoder (zxing-cpp, run
# through uv), which must return the same digits, so the symbols are readable
# by a scanner and not only by this package's tests. The ITF-14 symbols are
# rendered with every bearer bar style.
#
# Needs Go and uv; nothing is added to go.mod. Set PYTHON to another
# interpreter that has zxing-cpp and Pillow installed to skip uv.
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# A throwaway module outside the repository renders the symbols.
mkdir "$tmp/render"
cat > "$tmp/render/go.mod" <<EOM
module render
go 1.23
require github.com/galenzo17/gs1-go v0.0.0
replace github.com/galenzo17/gs1-go => $root
EOM
cat > "$tmp/render/main.go" <<'EOM'
package main

import (
	"fmt"
	"os"

	"github.com/galenzo17/gs1-go/symbol"
)

// usage: render SYMBOLOGY DATA SCALE BEARER OUT.png
func main() {
	var l *symbol.Linear
	var err error
	switch os.Args[1] {
	case "ean13":
		l, err = symbol.EAN13(os.Args[2])
	case "ean8":
		l, err = symbol.EAN8(os.Args[2])
	case "upca":
		l, err = symbol.UPCA(os.Args[2])
	case "upce":
		l, err = symbol.UPCE(os.Args[2])
	case "itf14":
		bearer := map[string]symbol.Bearer{"horizontal": symbol.BearerHorizontal, "frame": symbol.BearerFrame, "none": symbol.BearerNone}
		l, err = symbol.ITF14(os.Args[2], symbol.ITF14Options{Bearer: bearer[os.Args[4]]})
	default:
		err = fmt.Errorf("unknown symbology %q", os.Args[1])
	}
	var scale int
	if err == nil {
		_, err = fmt.Sscan(os.Args[3], &scale)
	}
	if err == nil {
		var f *os.File
		if f, err = os.Create(os.Args[5]); err == nil {
			err = l.PNG(f, scale)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
EOM
(cd "$tmp/render" && go build -o "$tmp/renderbin" .)

# symbology, data, digits zxing must return, zxing format names accepted.
# zxing-cpp reports a UPC-A as EAN-13 with a leading zero and a UPC-E as the
# UPC-A it expands to, so the expected digits are compared without leading
# zeros and UPC-E cases list the expanded UPC-A.
cases=(
	"ean13 4006381333931 4006381333931 EAN13"
	"ean13 0036000291452 0036000291452 EAN13"
	"ean13 9780201379624 9780201379624 EAN13"
	"ean13 5901234123457 5901234123457 EAN13"
	"ean8 96385074 96385074 EAN8"
	"ean8 49991130 49991130 EAN8"
	"upca 036000291452 036000291452 UPCA,EAN13"
	"upca 012345678905 012345678905 UPCA,EAN13"
	"upce 01234565 012345000065 UPCE"
	"upce 0123456 012345000065 UPCE"
	"upce 1123456 112345000062 UPCE"
	"upce 02345673 023456000073 UPCE"
	"itf14 10614141000019 10614141000019 ITF"
	"itf14 00012345678905 00012345678905 ITF"
	"itf14 98765432109213 98765432109213 ITF"
)
n=0
for c in "${cases[@]}"; do
	read -r sym data want format <<<"$c"
	bearers=(none)
	[ "$sym" = itf14 ] && bearers=(horizontal frame none)
	for bearer in "${bearers[@]}"; do
		for scale in 2 3 5; do
			"$tmp/renderbin" "$sym" "$data" "$scale" "$bearer" "$tmp/$n.png"
			printf '%s\t%s\t%s\t%s\n' "$tmp/$n.png" "$format" "$want" "$sym $data x$scale $bearer" >> "$tmp/list"
			n=$((n + 1))
		done
	done
done

read -ra py <<<"${PYTHON:-uv run --quiet --with zxing-cpp --with pillow python}"
"${py[@]}" - "$tmp/list" <<'EOF2'
import re, sys, zxingcpp
from PIL import Image

def norm(s):
    return re.sub(r"[^A-Z0-9]", "", str(s).upper())

formats = [zxingcpp.BarcodeFormat.EAN13, zxingcpp.BarcodeFormat.EAN8,
           zxingcpp.BarcodeFormat.UPCA, zxingcpp.BarcodeFormat.UPCE, zxingcpp.BarcodeFormat.ITF]
fail = total = 0
for line in open(sys.argv[1]):
    path, fmt, want, label = line.rstrip("\n").split("\t")
    total += 1
    got = zxingcpp.read_barcodes(Image.open(path), formats=formats)
    ok = (len(got) == 1 and got[0].text.lstrip("0") == want.lstrip("0")
          and norm(got[0].format) in fmt.split(",")
          and got[0].symbology_identifier.startswith("]I" if fmt == "ITF" else "]E"))
    if not ok:
        fail += 1
        print("FAIL", label, want, [(norm(g.format), g.symbology_identifier, g.text) for g in got])
    else:
        print("ok  ", label.ljust(36), got[0].symbology_identifier, got[0].text)
print("%d symbols, %d failed" % (total, fail))
sys.exit(1 if fail else 0)
EOF2
