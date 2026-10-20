/** A single AI-value pair from a GS1 barcode. */
export interface GS1Element {
  ai: string;
  /** Application Identifier name from the AI table, when known. */
  name?: string;
  value: string;
  /** Decoded measurement for four-digit measure AIs (310n, 320n, 330n, 340n). */
  measure?: GS1Measure;
  /** ISO 8601 date for AIs 11, 12, 13, 15 and 17 when `dateFormat` is "iso". */
  date?: string;
}

/** A numeric GS1 measurement with its original representation. */
export interface GS1Measure {
  /** Digits as encoded in the barcode. */
  raw: string;
  /** Value with the implied decimal point applied. */
  value: number;
  /** Integer value before applying `decimals` decimal places. */
  scaled: number;
  /** Number of implied decimal places, from the fourth AI digit. */
  decimals: number;
  /** Unit of measure. */
  unit: "kg" | "lb";
}

/** A non-fatal parsing advisory. */
export interface GS1Warning {
  code: string;
  message: string;
}

/** Result of parsing a GS1 barcode string. */
export interface GS1ParseResult {
  raw: string;
  elements: GS1Element[];
  gtin: string;
  lot: string;
  serial: string;
  /** Contained-item GTIN (AI 02). */
  contentGtin?: string;
  /** Count of trade items (AI 37). */
  countOfTradeItems?: string;
  /** Global Location Number (AI 414). */
  gln?: string;
  /** Global Shipment Identification Number (AI 402). */
  gsin?: string;
  /** Packaging date (AI 13). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  packagingDate?: string;
  /** Carrier name, "EPC" for RFID tags, or "unknown" when no recognized AIM prefix was present. */
  symbology: string;
  /** Decoded EPC identifier; only present on parseEPC results. */
  epc?: GS1EPC;
  /** Non-fatal parsing advisories. */
  warnings?: GS1Warning[];
  /** Expiration date (AI 17). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  expirationDate?: string;
  /** Production date (AI 11). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  productionDate?: string;
  /** Best-before date (AI 15). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  bestBeforeDate?: string;
}

/** Decoded EPC Tag Data Standard identifier attached to parseEPC results. */
export interface GS1EPC {
  /** Scheme name as used in EPC URIs: "sgtin", "sscc", "sgln", "grai" or "giai". */
  scheme: "sgtin" | "sscc" | "sgln" | "grai" | "giai";
  /** Tag length of the binary coding scheme (96, 198, 195, 170, 202); absent for pure-identity URIs. */
  bits?: number;
  /** Filter value of a tag encoding; absent for pure-identity URIs. */
  filter?: number;
  companyPrefix: string;
  /** Item reference with indicator digit, serial reference with extension digit, location reference, asset type or individual asset reference. */
  reference: string;
  /** SGTIN serial, SGLN extension or GRAI serial. */
  serial?: string;
  /** Pure-identity URI, e.g. "urn:epc:id:sgtin:0614141.812345.6789". */
  uri: string;
}

/** Error object returned when parsing fails. */
export interface GS1ParseError {
  error: string;
}

/** Result of gs1.encode(). */
export interface GS1EncodeResult {
  /** Scanner-equivalent element string; FNC1 is the ASCII GS character (0x1D). */
  raw: string;
  /** Human-readable interpretation in encoding order, e.g. "(01)04150000021126(10)LOT42". */
  hri: string;
  /** ZPL ^FD field data for ^BC in automatic mode, with >8 FNC1 escapes. */
  zpl: string;
}

/** Numeric GS1 key families that end in a modulo-10 check digit. */
export type GS1KeyType = "GTIN" | "SSCC" | "GLN" | "GSIN" | "GSRN";

/** Result of gs1.validateKey(). */
export interface GS1KeyResult {
  /** True when the length matches a key family and the check digit is correct. */
  valid: boolean;
  /**
   * Key families the length admits, in the order GTIN, SSCC, GLN, GSIN, GSRN.
   * GTIN-13 and GLN share 13 digits; SSCC and GSRN share 18. Empty on error.
   */
  keyTypes: GS1KeyType[];
  /** Reason the key is invalid: length, non-digit characters or check digit. */
  error?: string;
}

/** Options for gs1.parse(). */
export interface GS1ParseOptions {
  /** Treat unprefixed eight-digit numeric input as a bare GTIN-8 instead of AI data. */
  assumeBareGTIN8?: boolean;
  /** Validate Application Identifier association rules. */
  strict?: boolean;
  /**
   * Validate the check digit of every numeric GS1 key in the element string
   * (GTIN, SSCC, GLN family, GSIN, GSRN, GDTI and GRAI prefixes),
   * independently of strict.
   */
  validateCheckDigits?: boolean;
  /**
   * Date output format.
   * - "raw" (default): YYMMDD string as-is from the barcode
   * - "iso": ISO 8601 date string (YYYY-MM-DD)
   */
  dateFormat?: "raw" | "iso";
  /**
   * How to resolve day=00 in GS1 dates.
   * - "last" (default): last day of the month (GS1 standard)
   * - "first": first day of the month
   */
  dayZero?: "last" | "first";
}

declare global {
  namespace gs1 {
    /**
     * Parse a GS1 barcode string (GS1-128, DataMatrix, bracket notation).
     * Returns a parsed result object, or an object with an `error` field.
     */
    function parse(
      input: string,
      options?: GS1ParseOptions
    ): GS1ParseResult | GS1ParseError;

    /**
     * Build the element string for a barcode generator. Accepts an array of
     * {ai, value} objects, bracket notation such as
     * "(01)04150000021126(10)LOT42" (split literally, so the AIs are exactly
     * the ones written) or a raw scan string. Predefined-length AIs are emitted
     * first, FNC1 is placed where required, and association rules and
     * check digits are enforced. Returns an object with an `error` field
     * when the elements violate GS1 rules.
     */
    function encode(
      elements: GS1Element[] | string
    ): GS1EncodeResult | GS1ParseError;

    /**
     * Decode an RFID tag encoded with the EPC Tag Data Standard: hexadecimal
     * EPC memory content (SGTIN-96/198, SSCC-96, SGLN-96/195, GRAI-96/170,
     * GIAI-96/202), a tag URI or a pure-identity URI. Returns the same
     * shape as parse() with the equivalent GS1 elements and an `epc`
     * object, or an object with an `error` field.
     */
    function parseEPC(input: string): GS1ParseResult | GS1ParseError;

    /**
     * Validate a GTIN check digit.
     * Accepts GTIN-8, GTIN-12, GTIN-13, and GTIN-14.
     */
    function validateGTIN(gtin: string): boolean;

    /**
     * Validate the check digit of any numeric GS1 key (GTIN-8/12/13/14,
     * SSCC, GLN, GSIN, GSRN) and report the families its length admits.
     */
    function validateKey(key: string): GS1KeyResult;

    /**
     * Validate barcode against a LATAM regulator's requirements.
     * @param input - Raw barcode string
     * @param regulator - One of: "anvisa", "anmat", "snfa", "cofepris"
     * @returns null if compliant, or an error string
     */
    function validateRegulatory(
      input: string,
      regulator: "anvisa" | "anmat" | "snfa" | "cofepris"
    ): string | null;

    /**
     * Build an 18-digit SSCC from an extension digit, a GS1 company prefix
     * (4 to 12 digits) and a serial reference; together they must total 16
     * digits. Returns the SSCC, or an object with an `error` field.
     */
    function newSSCC(
      extensionDigit: string,
      companyPrefix: string,
      serialReference: string
    ): string | GS1ParseError;

    /**
     * Build a GTIN-13 from a company prefix and an item reference that
     * total 12 digits. Returns the GTIN-13, or an object with an `error` field.
     */
    function newGTIN13(
      companyPrefix: string,
      itemReference: string
    ): string | GS1ParseError;

    /**
     * Build a case-level GTIN-14 from an indicator digit and a valid GTIN-13.
     * Returns the GTIN-14, or an object with an `error` field. Every builder
     * argument must be a string; numbers and `{ error }` objects are rejected.
     */
    function newGTIN14(
      indicator: string,
      gtin13: string
    ): string | GS1ParseError;
  }
}
