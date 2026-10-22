// formula.ts types the rules module's static formula operations (docs/engineering/rules-engine.md,
// "The static formula operations"): formula.catalog, formula.check, formula.format,
// formula.signature and formula.identifier. The module is the formula language: it parses,
// checks and formats, and the Formula Studio's editor only draws what it answers. Every call
// is synchronous on a loaded runtime (loadRules(), rulesSync(), useRules()), and every position
// in and out is UTF-16 code units, which is what CodeMirror counts.
//
// The catalog is read once per loaded runtime and validated; every other answer comes from the
// same committed artifact the server runs and is typed, not re-validated, on each keystroke.

import {
  formulaCatalogSchema,
  type FormulaCatalog,
  type FormulaDiagnostic,
  type FormulaSchemaVariable,
  type FormulaValueType,
} from "@trenova/shared/types/formula-template";
import type { RulesRuntime } from "./runtime";

export const FORMULA_CATALOG_OP = "formula.catalog";
export const FORMULA_CHECK_OP = "formula.check";
export const FORMULA_FORMAT_OP = "formula.format";
export const FORMULA_SIGNATURE_OP = "formula.signature";
export const FORMULA_IDENTIFIER_OP = "formula.identifier";

/** What a formula must come to: an amount (the default) or a yes/no rule. */
export type FormulaExpect = "amount" | "predicate";

/** A template input as the checker reads it: `Number`, `String`, `Boolean`, any case. */
export type FormulaCheckInputDeclaration = { name: string; type: string };

/** A schema variable as the checker reads it; extra keys the endpoint publishes are ignored. */
export type FormulaCheckVariable = Pick<FormulaSchemaVariable, "name" | "type"> &
  Partial<Omit<FormulaSchemaVariable, "name" | "type">>;

/**
 * What the checker knows about names: the record's fields (a list's item fields named
 * `list[].field`), the template's inputs, and word operators beyond the built-in four.
 */
export type FormulaCheckSchema = {
  variables: readonly FormulaCheckVariable[];
  inputs: readonly FormulaCheckInputDeclaration[];
  operators?: readonly string[];
};

/** A rate table the formula may read, for the `rate-table` checks. */
export type FormulaRateTable = { name: string; axes: 1 | 2; banded?: boolean };

export type FormulaCheckRequest = {
  source: string;
  expect?: FormulaExpect;
  /** Null (or left out) while the schema has not loaded: the answer is `pending`. */
  schema?: FormulaCheckSchema | null;
  tables?: readonly FormulaRateTable[];
};

/** A top-level `let` line: its type, how often it is read, its range and its name's range. */
export type FormulaBinding = {
  name: string;
  type: FormulaValueType;
  reads: number;
  from: number;
  to: number;
  nameFrom: number;
  nameTo: number;
};

/** One `let` line or value of the formula, from its first token to the last of its value. */
export type FormulaCheckStatement = {
  index: number;
  kind: "let" | "result";
  name?: string;
  code: string;
  from: number;
  to: number;
};

/** A rate table the formula reads by a string literal, with the literal's range. */
export type FormulaLookupTable = { name: string; function: string; from: number; to: number };

/** What `formula.check` answers. A formula is accepted iff no diagnostic is an error. */
export type FormulaCheck = {
  /** Nothing but spaces and comments; there is then one `empty` error. */
  empty: boolean;
  /** The schema was not given, so unknown names and fields were not reported. */
  pending: boolean;
  /** Sorted by `from`, then `to`. */
  diagnostics: FormulaDiagnostic[];
  resultType: FormulaValueType;
  /** Every field and input read, inputs first, each in the order it first appears. */
  reads: string[];
  schemaReads: string[];
  inputReads: string[];
  /** The functions called; methods are not listed. */
  functionCalls: string[];
  bindings: FormulaBinding[];
  statements: FormulaCheckStatement[];
  lookupTables: FormulaLookupTable[];
};

export type FormulaFormatRequest = {
  source: string;
  /** The line width, in UTF-16 code units; 80 when left out. */
  width?: number;
  operators?: readonly string[];
};

/** Why the formatter left a formula alone: it would otherwise lose text. */
export type FormulaFormatRefusal = "empty" | "syntax" | "comments";

export type FormulaFormatResult =
  | { ok: true; text: string; changed: boolean }
  | { ok: false; reason: FormulaFormatRefusal };

export type FormulaCursorRequest = {
  source: string;
  /** The cursor, in UTF-16 code units. */
  position: number;
};

/** The innermost catalogued call around the cursor, the argument it is in, and its parentheses. */
export type FormulaSignature = { name: string; active: number; from: number; to: number };

export type FormulaIdentifierKind =
  | "function"
  | "method"
  | "field"
  | "let"
  | "pointer"
  | "input"
  | "variable"
  | "unknown";

/** The name under the cursor; a dotted path is one name. */
export type FormulaIdentifier = {
  name: string;
  kind: FormulaIdentifierKind;
  from: number;
  to: number;
};

const catalogs = new WeakMap<RulesRuntime, FormulaCatalog>();

/** Every function, built-in, disabled and reserved name; read once per loaded runtime. */
export function formulaCatalog(rules: RulesRuntime): FormulaCatalog {
  const cached = catalogs.get(rules);
  if (cached) return cached;
  const catalog = formulaCatalogSchema.parse(rules.call(FORMULA_CATALOG_OP, {}));
  catalogs.set(rules, catalog);
  return catalog;
}

/** Checks one formula: its findings with fixes, its types, what it reads and its statements. */
export function formulaCheck(rules: RulesRuntime, request: FormulaCheckRequest): FormulaCheck {
  return rules.call(FORMULA_CHECK_OP, {
    source: request.source,
    expect: request.expect ?? "amount",
    schema: request.schema ?? null,
    ...(request.tables ? { tables: request.tables } : {}),
  }) as FormulaCheck;
}

/** Formats one formula, or says why it was left alone. */
export function formulaFormat(
  rules: RulesRuntime,
  request: FormulaFormatRequest,
): FormulaFormatResult {
  return rules.call(FORMULA_FORMAT_OP, {
    source: request.source,
    ...(request.width !== undefined ? { width: request.width } : {}),
    operators: request.operators ?? [],
  }) as FormulaFormatResult;
}

/** The call the cursor is inside and the argument it is on; null outside any call. */
export function formulaSignature(
  rules: RulesRuntime,
  request: FormulaCursorRequest & { operators?: readonly string[] },
): FormulaSignature | null {
  return rules.call(FORMULA_SIGNATURE_OP, {
    source: request.source,
    position: request.position,
    operators: request.operators ?? [],
  }) as FormulaSignature | null;
}

/**
 * The identifier under the cursor and what it is; null when the cursor is on no name. Without a
 * schema every name that is no function, method, field, `let` or pointer is a `variable`.
 */
export function formulaIdentifier(
  rules: RulesRuntime,
  request: FormulaCursorRequest & { schema?: FormulaCheckSchema | null },
): FormulaIdentifier | null {
  return rules.call(FORMULA_IDENTIFIER_OP, {
    source: request.source,
    position: request.position,
    schema: request.schema ?? null,
  }) as FormulaIdentifier | null;
}
