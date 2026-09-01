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
};

export type KnownIdentifiers = {
  variables: VariableDoc[];
  functions: FunctionDoc[];
  variableRoots: Set<string>;
  variablePaths: Set<string>;
  functionNames: Set<string>;
};

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
  }));

  const variableRoots = new Set<string>();
  const variablePaths = new Set<string>();
  for (const variable of variables) {
    variablePaths.add(variable.name);
    const dot = variable.name.indexOf(".");
    variableRoots.add(dot === -1 ? variable.name : variable.name.slice(0, dot));
  }

  const functionNames = new Set<string>(functions.map((fn) => fn.name));

  return { variables, functions, variableRoots, variablePaths, functionNames };
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
