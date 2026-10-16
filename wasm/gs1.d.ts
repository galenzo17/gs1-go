/** A single AI-value pair from a GS1 barcode. */
export interface GS1Element {
  ai: string;
  value: string;
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

/** Options for gs1.parse(). */
export interface GS1ParseOptions {
  /** Validate Application Identifier association rules. */
  strict?: boolean;
  /** Validate GTIN check digits for AI 01 and AI 02, independently of strict. */
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
     * Validate barcode against a LATAM regulator's requirements.
     * @param input - Raw barcode string
     * @param regulator - One of: "anvisa", "anmat", "snfa", "cofepris"
     * @returns null if compliant, or an error string
     */
    function validateRegulatory(
      input: string,
      regulator: "anvisa" | "anmat" | "snfa" | "cofepris"
    ): string | null;
  }
}
