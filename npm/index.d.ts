/** A column definition: the kind, plus whatever params that kind takes. */
export interface Column {
  kind: string;
  /** Fraction of rows left empty, "0".."100". */
  blank?: string | number;
  /** partial | hash | redact | token */
  mask?: 'partial' | 'hash' | 'redact' | 'token';
  /** Salt for mask=hash and mask=token. */
  salt?: string;
  /** HMAC key for mask=hash. Without it a short value can be guessed and hashed until it matches. */
  secret?: string;
  [param: string]: unknown;
}

export interface GenerateOptions {
  /** A built-in schema name; see listPresets(). Use this or schema, not both. */
  preset?: string;
  /** Columns as {name: {kind, ...}}. Use this or preset, not both. */
  schema?: Record<string, Column>;
  /** How many rows. Default 10. */
  rows?: number;
  /** Data locale, e.g. "uz_UZ". Default "en_US". */
  locale?: string;
  /** The same seed always produces the same rows. */
  seed?: number;
  /** Return raw card numbers and identifiers. Off by default. */
  unmasked?: boolean;
  /** Table name, used as the SQL table and the CSV file's subject. */
  name?: string;
}

export interface TypeInfo {
  kind: string;
  category: string;
  /** Whether the values change with the locale. Most types do not. */
  localized: boolean;
  locales?: string[];
}

export interface PresetInfo {
  name: string;
  /** The preset's YAML, so it can be read and edited rather than guessed at. */
  yaml: string;
}

/**
 * Generate records.
 *
 * Card numbers and national identifiers come back masked unless `unmasked` is
 * set — a fixture reaches a ticket or a log sooner or later, and the safe form
 * being the default means nobody has to remember.
 */
export function generate(options?: GenerateOptions): Promise<Record<string, unknown>[]>;

/** Generate and serialise in one step, for writing a fixture straight to disk. */
export function generateAs(
  format: 'csv' | 'jsonl' | 'sql' | 'json',
  options?: GenerateOptions,
): Promise<string>;

/** The generatable column types. */
export function listTypes(): Promise<TypeInfo[]>;

/** The available data locales, sorted. */
export function listLocales(): Promise<string[]>;

/** The built-in schemas, each with its YAML. */
export function listPresets(): Promise<PresetInfo[]>;

/** The version of the compiled generator, for a bug report. */
export function version(): Promise<string>;
