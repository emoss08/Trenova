import {
  AVAILABLE_FUNCTIONS,
  SHIPMENT_VARIABLES,
  type FormulaSchemaResponse,
  type VariableDefinitionInput,
} from "@trenova/shared/types/formula-template";

export type VariableDoc = {
  name: string;
  type: string;
  description: string;
  category: string;
  nullable?: boolean;
  custom?: boolean;
  enum?: string[] | null;
};

export type FunctionDoc = {
  name: string;
  signature: string;
  description: string;
  example?: string;
  category?: string;
  /** Written infix (`text startsWith "x"`) or as a slice, not as a call. */
  operator?: boolean;
};

export type KnownIdentifiers = {
  variables: VariableDoc[];
  functions: FunctionDoc[];
  variableRoots: Set<string>;
  variablePaths: Set<string>;
  functionNames: Set<string>;
  operatorNames: Set<string>;
};

/**
 * Words expr parses as binary operators. They look like identifiers to a
 * tokenizer, so the linter and highlighter must not treat them as variables.
 */
export const EXPR_OPERATOR_WORDS = new Set(["startsWith", "endsWith", "contains", "matches"]);

export const EXPR_KEYWORDS = new Set([
  "true",
  "false",
  "nil",
  "in",
  "not",
  "and",
  "or",
  "let",
  "if",
  "else",
]);

/**
 * Functions the expr runtime ships beyond Trenova's own. The linter must not
 * flag a call to these even though the reference panel does not list them.
 */
export const EXPR_BUILTIN_FUNCTIONS = new Set([
  "all",
  "any",
  "one",
  "none",
  "map",
  "filter",
  "find",
  "findIndex",
  "findLast",
  "findLastIndex",
  "count",
  "concat",
  "flatten",
  "uniq",
  "join",
  "reduce",
  "mean",
  "median",
  "first",
  "last",
  "take",
  "sortBy",
  "sort",
  "groupBy",
  "len",
  "int",
  "float",
  "string",
  "trim",
  "trimPrefix",
  "trimSuffix",
  "upper",
  "lower",
  "split",
  "splitAfter",
  "replace",
  "repeat",
  "indexOf",
  "lastIndexOf",
  "hasPrefix",
  "hasSuffix",
  "toJSON",
  "fromJSON",
  "toBase64",
  "fromBase64",
  "now",
  "duration",
  "date",
  "timezone",
  "get",
  "keys",
  "values",
  "type",
  "lookup",
  "lookupOr",
  "lookup2",
  "lookup2Or",
  "lookupInterp",
  "deficitWeight",
]);

export const FALLBACK_SCHEMA: Pick<FormulaSchemaResponse, "variables" | "functions"> = {
  variables: SHIPMENT_VARIABLES.map((variable) => ({
    name: variable.name,
    type: variable.type,
    description: variable.description,
    category: variable.category,
    nullable: variable.nullable ?? false,
    computed: variable.category === "computed",
    enum: undefined,
  })),
  functions: AVAILABLE_FUNCTIONS.map((fn) => ({
    name: fn.name,
    signature: fn.signature,
    description: fn.description,
    example: "",
    category: "",
    operator: false,
  })),
};

export function buildKnownIdentifiers(
  schema: Pick<FormulaSchemaResponse, "variables" | "functions"> | undefined,
  customVariables: VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const source = schema && schema.variables.length > 0 ? schema : FALLBACK_SCHEMA;

  const variables: VariableDoc[] = source.variables.map((variable) => ({
    name: variable.name,
    type: variable.type,
    description: variable.description ?? "",
    category: variable.category ?? "",
    nullable: variable.nullable ?? false,
    enum: variable.enum ?? null,
  }));

  for (const custom of customVariables) {
    if (!custom.name) continue;
    variables.push({
      name: custom.name,
      type: custom.type ?? "Any",
      description: custom.description || "Custom variable",
      category: "custom",
      custom: true,
    });
  }

  const functions: FunctionDoc[] = source.functions.map((fn) => ({
    name: fn.name,
    signature: fn.signature,
    description: fn.description ?? "",
    example: fn.example || undefined,
    category: fn.category || undefined,
    operator: fn.operator || undefined,
  }));

  const variableRoots = new Set<string>();
  const variablePaths = new Set<string>();
  for (const variable of variables) {
    variablePaths.add(variable.name);
    const dot = variable.name.indexOf(".");
    variableRoots.add(dot === -1 ? variable.name : variable.name.slice(0, dot));
  }

  const functionNames = new Set<string>();
  const operatorNames = new Set<string>(EXPR_OPERATOR_WORDS);
  for (const fn of functions) {
    if (fn.operator) {
      operatorNames.add(fn.name);
    } else {
      functionNames.add(fn.name);
    }
  }

  return { variables, functions, variableRoots, variablePaths, functionNames, operatorNames };
}

export function isOperatorWord(known: KnownIdentifiers, identifier: string): boolean {
  return known.operatorNames.has(identifier);
}

export type FunctionInsertion = { text: string; cursor: number };

/**
 * What clicking or completing a function drops into the editor. Calls get
 * empty parentheses with the cursor inside; infix operators get surrounding
 * spaces and an empty string to type into; a slice gets a sample range with
 * the cursor on the start index.
 */
export function functionInsertion(
  fn: Pick<FunctionDoc, "name"> & Partial<FunctionDoc>,
): FunctionInsertion {
  if (!fn.operator) {
    const text = `${fn.name}()`;
    return { text, cursor: text.length - 1 };
  }
  if (fn.name.startsWith("[")) {
    return { text: "[0:3]", cursor: 1 };
  }
  const text = ` ${fn.name} ""`;
  return { text, cursor: text.length - 1 };
}

export function isKnownVariable(known: KnownIdentifiers, identifier: string): boolean {
  if (known.variablePaths.has(identifier)) return true;
  const dot = identifier.indexOf(".");
  const root = dot === -1 ? identifier : identifier.slice(0, dot);
  return known.variableRoots.has(root);
}

export function isKnownFunction(known: KnownIdentifiers, identifier: string): boolean {
  return known.functionNames.has(identifier) || EXPR_BUILTIN_FUNCTIONS.has(identifier);
}
