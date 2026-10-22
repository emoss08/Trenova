import { translate } from "@trenova/shared/i18n/runtime";
import type { FormulaCheckSchema } from "@trenova/shared/rules/formula";
import type {
  FormulaCatalog,
  FormulaDisabledFunction,
  FormulaFunction,
  FormulaSchemaResponse,
  VariableDefinitionInput,
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

/** A function as `formula.catalog` lists it; its description and example are messages. */
export type FunctionDoc = FormulaFunction;

/**
 * What the editor knows about names, from two sources: the record's fields as
 * `GET /formula-templates/schema` publishes them (with the template's inputs),
 * and the functions, built-ins, keywords and disabled names of the rules
 * module's `formula.catalog`. The checks themselves are the module's; this is
 * what the highlighter, the completions, the hover card and the reference read.
 */
export type KnownIdentifiers = {
  /** False until the schema has arrived; until then no name is called unknown. */
  ready: boolean;
  /** The record's fields and the template's inputs, as the reference lists them. */
  variables: VariableDoc[];
  /** The fields of a list's items, named `list[].field` as the schema publishes them. */
  itemFields: VariableDoc[];
  /** The documented functions and word operators the reference lists. */
  functions: FunctionDoc[];
  /** Every function a formula calls by name, documented or built-in. */
  callables: ReadonlyMap<string, FunctionDoc>;
  disabled: ReadonlyMap<string, FormulaDisabledFunction>;
  /** The language's keywords: every reserved word that is not a function. */
  keywords: ReadonlySet<string>;
  /** The words written as operators (`text startsWith "x"`). */
  operatorNames: ReadonlySet<string>;
  variableRoots: ReadonlySet<string>;
  variablePaths: ReadonlySet<string>;
  /** What `formula.check` and `formula.identifier` read; null until the schema has arrived. */
  schema: FormulaCheckSchema | null;
};

type CatalogKnowledge = Pick<
  KnownIdentifiers,
  "functions" | "callables" | "disabled" | "keywords" | "operatorNames"
>;

const ITEM_FIELD = "[].";
const WORD = /^[A-Za-z_]\w*$/;
const fromCatalog = new WeakMap<FormulaCatalog, CatalogKnowledge>();

function catalogKnowledge(catalog: FormulaCatalog): CatalogKnowledge {
  const cached = fromCatalog.get(catalog);
  if (cached) return cached;
  const callables = new Map<string, FunctionDoc>();
  const operatorNames = new Set<string>();
  for (const fn of catalog.functions) {
    if (fn.operator) {
      if (WORD.test(fn.name)) operatorNames.add(fn.name);
    } else {
      callables.set(fn.name, fn);
    }
  }
  for (const fn of catalog.builtins) callables.set(fn.name, fn);
  const knowledge: CatalogKnowledge = {
    functions: catalog.functions,
    callables,
    disabled: new Map(catalog.disabled.map((fn) => [fn.name, fn])),
    keywords: new Set(
      catalog.reserved.filter((word) => !callables.has(word) && !operatorNames.has(word)),
    ),
    operatorNames,
  };
  fromCatalog.set(catalog, knowledge);
  return knowledge;
}

function rootOf(name: string): string {
  const dot = name.indexOf(".");
  return dot === -1 ? name : name.slice(0, dot);
}

/**
 * Builds what the editor knows from the module's catalog, the schema (undefined
 * while it loads) and the template's inputs.
 */
export function buildKnownIdentifiers(
  catalog: FormulaCatalog,
  schema: Pick<FormulaSchemaResponse, "variables"> | undefined,
  customVariables: readonly VariableDefinitionInput[] = [],
): KnownIdentifiers {
  const variables: VariableDoc[] = [];
  const itemFields: VariableDoc[] = [];
  for (const variable of schema?.variables ?? []) {
    const doc: VariableDoc = {
      name: variable.name,
      type: variable.type,
      description: variable.description ?? "",
      category: variable.category ?? "",
      nullable: variable.nullable ?? false,
      enum: variable.enum ?? null,
    };
    (variable.name.includes(ITEM_FIELD) ? itemFields : variables).push(doc);
  }

  const inputs = customVariables.filter((custom) => custom.name);
  for (const custom of inputs) {
    variables.push({
      name: custom.name,
      type: custom.type ?? "Any",
      description: custom.description || translate("Custom variable"),
      category: "custom",
      custom: true,
    });
  }

  const variableRoots = new Set<string>();
  const variablePaths = new Set<string>();
  for (const variable of variables) {
    variablePaths.add(variable.name);
    variableRoots.add(rootOf(variable.name));
  }

  return {
    ...catalogKnowledge(catalog),
    ready: schema !== undefined,
    variables,
    itemFields,
    variableRoots,
    variablePaths,
    schema: schema
      ? {
          variables: schema.variables,
          inputs: inputs.map((custom) => ({ name: custom.name, type: custom.type ?? "Any" })),
        }
      : null,
  };
}

export function isOperatorWord(known: KnownIdentifiers, identifier: string): boolean {
  return known.operatorNames.has(identifier);
}

export function isKeyword(known: KnownIdentifiers, identifier: string): boolean {
  return known.keywords.has(identifier);
}

export type FunctionInsertion = { text: string; cursor: number };

/**
 * What clicking or completing a function drops into the editor. Calls get
 * empty parentheses with the cursor inside; infix operators get surrounding
 * spaces and an empty string to type into; a slice gets a sample range with
 * the cursor on the start index.
 */
export function functionInsertion(fn: Pick<FunctionDoc, "name" | "operator">): FunctionInsertion {
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
  return known.variableRoots.has(rootOf(identifier));
}

export function isKnownFunction(known: KnownIdentifiers, identifier: string): boolean {
  return known.callables.has(identifier);
}

/** The schema's description of an item's field (`.weight`), when every list that has it agrees. */
export function itemFieldDoc(known: KnownIdentifiers, field: string): VariableDoc | null {
  const suffix = `${ITEM_FIELD}${field}`;
  const matches = known.itemFields.filter((doc) => doc.name.endsWith(suffix));
  const [first] = matches;
  if (!first) return null;
  const agree = matches.every(
    (doc) => doc.type === first.type && Boolean(doc.nullable) === Boolean(first.nullable),
  );
  return agree ? { ...first, name: field } : null;
}
