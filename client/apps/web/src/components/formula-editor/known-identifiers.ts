import type {
  FormulaDisabledReason,
  FormulaSchemaResponse,
  FormulaValueType,
  VariableDefinitionInput,
} from "@trenova/shared/types/formula-template";
import { VARIADIC_ARGS } from "@trenova/shared/types/formula-template";
import { KEYWORDS, WORD_OPERATORS } from "./expr-parser";

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

/** How a function is called and what it gives back, as the engine publishes it. */
export type FunctionShape = {
  name: string;
  signature: string;
  minArgs: number;
  /** Infinity when it takes any number of arguments. */
  maxArgs: number;
  returns: FormulaValueType;
  numericArgs: boolean;
  /** Its second argument is evaluated per item, where `#` and `.field` read the item. */
  predicate: boolean;
};

export type DisabledFunction = {
  name: string;
  reason: FormulaDisabledReason;
  replacement: string | null;
};

export type KnownIdentifiers = {
  /** False until the engine's schema has arrived; nothing can be called unknown before then. */
  ready: boolean;
  variables: VariableDoc[];
  /** The documented functions the reference lists. */
  functions: FunctionDoc[];
  /** Every callable name, documented or a builtin the engine keeps, with its shape. */
  shapes: Map<string, FunctionShape>;
  disabled: Map<string, DisabledFunction>;
  variableRoots: Set<string>;
  variablePaths: Set<string>;
  functionNames: Set<string>;
  operatorNames: Set<string>;
};

/** Words expr parses as binary operators; a tokenizer must not take them for variables. */
export const EXPR_OPERATOR_WORDS: ReadonlySet<string> = WORD_OPERATORS;

export const EXPR_KEYWORDS: ReadonlySet<string> = KEYWORDS;

type SchemaSource = Pick<FormulaSchemaResponse, "variables" | "functions"> &
  Partial<Pick<FormulaSchemaResponse, "builtins" | "disabled">>;

function toShape(fn: FormulaSchemaResponse["functions"][number]): FunctionShape {
  return {
    name: fn.name,
    signature: fn.signature,
    minArgs: fn.minArgs,
    maxArgs: fn.maxArgs === VARIADIC_ARGS ? Number.POSITIVE_INFINITY : fn.maxArgs,
    returns: fn.returns,
    numericArgs: fn.numericArgs,
    predicate: fn.predicate,
  };
}

/**
 * What the editor knows about names, from the engine's own description of the
 * schema (`GET /formula-templates/schema`) and the template's inputs. Until the
 * description has loaded, only the inputs are known and `ready` is false.
 */
export function buildKnownIdentifiers(
  schema: SchemaSource | undefined,
  customVariables: VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const variables: VariableDoc[] = (schema?.variables ?? []).map((variable) => ({
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

  const functions: FunctionDoc[] = (schema?.functions ?? []).map((fn) => ({
    name: fn.name,
    signature: fn.signature,
    description: fn.description ?? "",
    example: fn.example || undefined,
    category: fn.category || undefined,
    operator: fn.operator || undefined,
  }));

  const shapes = new Map<string, FunctionShape>();
  const functionNames = new Set<string>();
  const operatorNames = new Set<string>(EXPR_OPERATOR_WORDS);
  for (const fn of schema?.functions ?? []) {
    if (fn.operator) {
      operatorNames.add(fn.name);
      continue;
    }
    functionNames.add(fn.name);
    shapes.set(fn.name, toShape(fn));
  }
  for (const fn of schema?.builtins ?? []) {
    functionNames.add(fn.name);
    shapes.set(fn.name, toShape(fn));
  }

  const disabled = new Map<string, DisabledFunction>(
    (schema?.disabled ?? []).map((fn) => [
      fn.name,
      { name: fn.name, reason: fn.reason, replacement: fn.replacement ?? null },
    ]),
  );

  const variableRoots = new Set<string>();
  const variablePaths = new Set<string>();
  for (const variable of variables) {
    variablePaths.add(variable.name);
    const dot = variable.name.indexOf(".");
    variableRoots.add(dot === -1 ? variable.name : variable.name.slice(0, dot));
  }

  return {
    ready: schema !== undefined,
    variables,
    functions,
    shapes,
    disabled,
    variableRoots,
    variablePaths,
    functionNames,
    operatorNames,
  };
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
  return known.functionNames.has(identifier);
}
