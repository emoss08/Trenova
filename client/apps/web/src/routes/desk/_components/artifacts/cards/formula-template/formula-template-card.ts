import {
  EXPR_KEYWORDS,
  isKnownFunction,
  isKnownVariable,
  isOperatorWord,
  type KnownIdentifiers,
} from "@/components/formula-editor/known-identifiers";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  FormulaRoundingMode,
  FormulaTemplateStatus,
  FormulaTemplateType,
  GuardrailResult,
  TemplateUsageCount,
  VariableDefinition,
  VariableValueType,
} from "@trenova/shared/types/formula-template";
import { money } from "../../record-card/record-card-values";

/**
 * What a piece of a formula is, as the card colours it: the template's own
 * variables apart from the shipment's fields, so a reader sees at a glance
 * which figures the template sets and which the load brings.
 */
export type FormulaTokenKind =
  | "var"
  | "field"
  | "fn"
  | "kw"
  | "op"
  | "num"
  | "str"
  | "comment"
  | "ident"
  | "plain";

export type FormulaToken = { text: string; kind: FormulaTokenKind };

const NUMBER = /^(?:0[xX][0-9a-fA-F_]+|\d[\d_]*(?:\.\d[\d_]*)?(?:[eE][+-]?\d+)?|\.\d[\d_]*)/u;
const STRING = /^(?:"(?:[^"\\]|\\.)*"?|'(?:[^'\\]|\\.)*'?|`(?:[^`\\]|\\.)*`?)/u;
const IDENTIFIER = /^[A-Za-z_$][\w$]*(?:\??\.[A-Za-z_$][\w$]*)*/u;
const OPERATOR = /^(?:\*\*|==|!=|<=|>=|&&|\|\||\?\?|\?\.|\.\.|[-+*/%<>!?:|^])/u;
const SPACE = /^\s+/u;

/**
 * A formula cut into the pieces the card colours. Every character of the
 * expression lands in exactly one token, in order, so joining the tokens
 * gives the expression back as it was written.
 */
export function tokenizeFormula(
  expression: string,
  known: KnownIdentifiers,
  templateVariables: ReadonlySet<string>,
): FormulaToken[] {
  const tokens: FormulaToken[] = [];
  let rest = expression;

  const push = (text: string, kind: FormulaTokenKind) => {
    const last = tokens.at(-1);
    if (last && last.kind === kind && (kind === "plain" || kind === "comment")) {
      last.text += text;
      return;
    }
    tokens.push({ text, kind });
  };

  while (rest !== "") {
    const space = SPACE.exec(rest);
    if (space) {
      push(space[0], "plain");
      rest = rest.slice(space[0].length);
      continue;
    }
    if (rest.startsWith("//")) {
      const end = rest.indexOf("\n");
      const text = end === -1 ? rest : rest.slice(0, end);
      push(text, "comment");
      rest = rest.slice(text.length);
      continue;
    }
    if (rest.startsWith("/*")) {
      const end = rest.indexOf("*/", 2);
      const text = end === -1 ? rest : rest.slice(0, end + 2);
      push(text, "comment");
      rest = rest.slice(text.length);
      continue;
    }
    const string = STRING.exec(rest);
    if (string) {
      push(string[0], "str");
      rest = rest.slice(string[0].length);
      continue;
    }
    const number = NUMBER.exec(rest);
    if (number) {
      push(number[0], "num");
      rest = rest.slice(number[0].length);
      continue;
    }
    const identifier = IDENTIFIER.exec(rest);
    if (identifier) {
      const text = identifier[0];
      rest = rest.slice(text.length);
      push(text, identifierKind(text, rest, known, templateVariables));
      continue;
    }
    const operator = OPERATOR.exec(rest);
    if (operator) {
      push(operator[0], "op");
      rest = rest.slice(operator[0].length);
      continue;
    }
    push(rest[0], "plain");
    rest = rest.slice(1);
  }

  return tokens;
}

function identifierKind(
  text: string,
  after: string,
  known: KnownIdentifiers,
  templateVariables: ReadonlySet<string>,
): FormulaTokenKind {
  if (EXPR_KEYWORDS.has(text)) return "kw";
  if (isOperatorWord(known, text)) return "op";
  if (/^\s*\(/u.test(after) && isKnownFunction(known, text)) return "fn";
  if (templateVariables.has(text) || templateVariables.has(rootOf(text))) return "var";
  if (isKnownVariable(known, text)) return "field";
  return /^\s*\(/u.test(after) ? "fn" : "ident";
}

/** The variable a token names: "customer.name" is the customer's, "baseRate" its own. */
export function rootOf(identifier: string): string {
  const dot = identifier.search(/\??\./u);
  return dot === -1 ? identifier : identifier.slice(0, dot);
}

/** Where a template stands, as the card colours it. */
export type FormulaStage = "active" | "review" | "draft" | "inactive";

export function formulaStage(status: FormulaTemplateStatus): FormulaStage {
  switch (status) {
    case "Active":
      return "active";
    case "InReview":
      return "review";
    case "Inactive":
      return "inactive";
    default:
      return "draft";
  }
}

export function formulaStatusLabel(status: FormulaTemplateStatus, t: TranslateFn): string {
  switch (status) {
    case "Active":
      return t("Active");
    case "InReview":
      return t("In review");
    case "Inactive":
      return t("Inactive");
    default:
      return t("Draft");
  }
}

/** Why the status matters to a biller, in the words the card puts beside it. */
export function formulaStatusNote(status: FormulaTemplateStatus, t: TranslateFn): string {
  switch (status) {
    case "Active":
      return t("Prices shipments");
    case "InReview":
      return t("Waiting on approval before it prices shipments");
    case "Inactive":
      return t("Retired; prices nothing new");
    default:
      return t("Not yet submitted for approval");
  }
}

export function formulaTypeLabel(type: FormulaTemplateType, t: TranslateFn): string {
  return type === "AccessorialCharge" ? t("Accessorial charge") : t("Freight charge");
}

export function roundingLabel(
  mode: FormulaRoundingMode,
  precision: number,
  t: TranslateFn,
): string {
  switch (mode) {
    case "None":
      return t("Not rounded");
    case "HalfEven":
      return t(
        "{0, plural, one {Half to even, # place} other {Half to even, # places}}",
        precision,
      );
    case "Up":
      return t("{0, plural, one {Up, # place} other {Up, # places}}", precision);
    case "Down":
      return t("{0, plural, one {Down, # place} other {Down, # places}}", precision);
    default:
      return t("{0, plural, one {Half up, # place} other {Half up, # places}}", precision);
  }
}

/** What rounding did to a price: the figure before it, to the places the engine kept. */
export function roundingNote(unrounded: number, t: TranslateFn): string {
  return t(
    "Rounded from {0}",
    unrounded.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 6 }),
  );
}

/** What uses a template, counted the way a person says it. */
export function usageLabel(usage: TemplateUsageCount, t: TranslateFn): string {
  switch (usage.type) {
    case "shipment":
      return t("{0, plural, one {# shipment} other {# shipments}}", usage.count);
    case "rate_matrix":
      return t("{0, plural, one {# rate matrix} other {# rate matrices}}", usage.count);
    case "rate_agreement_rule":
      return t("{0, plural, one {# agreement rule} other {# agreement rules}}", usage.count);
    case "rate_agreement_accessorial":
      return t(
        "{0, plural, one {# agreement accessorial} other {# agreement accessorials}}",
        usage.count,
      );
    default:
      return t("{0, plural, one {# other use} other {# other uses}}", usage.count);
  }
}

/** What the minimum or maximum did to a price, when it did anything. */
export function guardrailNote(guardrail: GuardrailResult | null | undefined, t: TranslateFn) {
  if (!guardrail?.applied) return "";
  return guardrail.bound === "max"
    ? t("Held to the maximum, down from {0}", money(guardrail.rawAmount))
    : t("Raised to the minimum, up from {0}", money(guardrail.rawAmount));
}

export function variableTypeLabel(type: VariableValueType, t: TranslateFn): string {
  switch (type) {
    case "Number":
      return t("Number");
    case "String":
      return t("Text");
    case "Boolean":
      return t("Yes or no");
    case "Date":
      return t("Date");
    case "Array":
      return t("List");
    case "Object":
      return t("Record");
    default:
      return t("Any");
  }
}

/** Whether the card can take a value for a variable in a plain field. */
export function isEditableVariable(type: VariableValueType): boolean {
  return type === "Number" || type === "String" || type === "Boolean" || type === "Any";
}

/** A variable's default as the field shows it before anyone changes it. */
export function variableDraft(definition: VariableDefinition): string {
  const value = definition.defaultValue;
  if (value === undefined || value === null) return "";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number" || typeof value === "string") return String(value);
  return JSON.stringify(value);
}

/** Every variable's field as it stands before anyone changes it, by variable. */
export function variableDrafts(definitions: readonly VariableDefinition[]): Record<string, string> {
  return Object.fromEntries(
    definitions.map((definition) => [definition.name, variableDraft(definition)]),
  );
}

/**
 * What a field holds, as the engine takes it: a number for a number, a yes or
 * no for a yes or no. An empty field, or a figure that does not read as one,
 * leaves the default in place rather than sending something the formula
 * cannot use.
 */
export function variableValue(type: VariableValueType, text: string): unknown {
  const trimmed = text.trim();
  if (trimmed === "") return undefined;
  switch (type) {
    case "Number": {
      const figure = Number(trimmed.replaceAll(",", ""));
      return Number.isFinite(figure) ? figure : undefined;
    }
    case "Boolean":
      return trimmed === "true" ? true : trimmed === "false" ? false : undefined;
    case "Any": {
      const figure = Number(trimmed.replaceAll(",", ""));
      return Number.isFinite(figure) ? figure : trimmed;
    }
    default:
      return trimmed;
  }
}

/** The values a person changed, as the engine takes them, by variable. */
export function variableOverrides(
  definitions: readonly VariableDefinition[],
  drafts: Readonly<Record<string, string>>,
): Record<string, unknown> {
  const overrides: Record<string, unknown> = {};
  for (const definition of definitions) {
    const draft = drafts[definition.name];
    if (draft === undefined || !isEditableVariable(definition.type)) continue;
    const value = variableValue(definition.type, draft);
    if (value !== undefined) {
      overrides[definition.name] = value;
    }
  }
  return overrides;
}

/**
 * The sample load a formula is priced on, as a biller pictures it: where it
 * runs, how far and how heavy. A part the values leave out is left out.
 */
export function sampleLoadSummary(values: Readonly<Record<string, unknown>>, t: TranslateFn) {
  const place = (prefix: string) =>
    [values[`${prefix}.city`], values[`${prefix}.state`]]
      .filter((part) => typeof part === "string" && part !== "")
      .join(", ");
  const from = place("origin");
  const to = place("destination");
  const miles = Number(values.totalDistance);
  const pounds = Number(values.totalWeight);
  return [
    from && to ? t("{0} to {1}", from, to) : "",
    Number.isFinite(miles) && miles > 0 ? t("{0} mi", miles.toLocaleString()) : "",
    Number.isFinite(pounds) && pounds > 0 ? t("{0} lb", pounds.toLocaleString()) : "",
  ].filter((part) => part !== "");
}
