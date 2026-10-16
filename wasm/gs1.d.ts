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
  /** Carrier name, or "unknown" when no recognized AIM prefix was present. */
  symbology: string;
  /** Non-fatal parsing advisories. */
  warnings?: GS1Warning[];
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
  }
}
