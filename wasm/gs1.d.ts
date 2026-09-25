/** A single AI-value pair from a GS1 barcode. */
export interface GS1Element {
  ai: string;
  value: string;
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
  /** Carrier name, or "unknown" when no recognized AIM prefix was present. */
  symbology: string;
  /** Expiration date (AI 17). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  expirationDate?: string;
  /** Production date (AI 11). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  productionDate?: string;
  /** Best-before date (AI 15). Raw YYMMDD or ISO 8601 depending on dateFormat. */
  bestBeforeDate?: string;
}

/** Error object returned when parsing fails. */
export interface GS1ParseError {
  error: string;
}

/** Options for gs1.parse(). */
export interface GS1ParseOptions {
  /** Validate Application Identifier association rules. */
  strict?: boolean;
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
