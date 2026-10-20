# Logistic label element strings

`logistic-labels.txt` holds element strings as the common carriers deliver
them for the GS1 logistic label (SSCC, CONTENT, COUNT, BATCH/LOT, dates):
GS1-128 through a keyboard wedge with and without the `]C1` symbology
identifier, GS1 DataMatrix with `]d2`, GS1 QR with `]Q3`, and the bracket
notation printed as HRI. Each line is `<carrier>\t<data>` with FNC1 written
as `{GS}`; `TestLogisticLabelFixtures` parses every line with association
and check-digit validation enabled and checks that all carriers yield the
same elements.

The GS1 verifier reports for printed samples (ISO/IEC 15416 for GS1-128,
ISO/IEC 15415 for GS1 DataMatrix) are recorded here once the renderers from
#24 produce the samples and the local GS1 member organisation checks them.
Until then this directory documents data conformance only.
