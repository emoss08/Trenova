import { formatExpression } from "@/components/formula-editor/expr-format";
import type {
  FormulaDraftBreakdown,
  FormulaDraftVariable,
  FormulaPolicy,
  FormulaProposal,
  PageDraft,
} from "@/types/page-draft";
import { limitCodePoints } from "@trenova/shared/lib/utils";
import type {
  FormulaRoundingMode,
  VariableDefinition,
  VariableValueType,
} from "@trenova/shared/types/formula-template";

/* The bounds pagedraft.Formula enforces; past any of them the turn is refused. */
const MAX_EXPRESSION_LENGTH = 10_000;
const MAX_VARIABLES = 50;
const MAX_VARIABLE_NAME_LENGTH = 64;
const MAX_VARIABLE_TEXT_LENGTH = 500;
const MAX_TEMPLATE_NAME_LENGTH = 100;
const MAX_BREAKDOWN_LINES = 20;
const MAX_BREAKDOWN_LABEL = 100;
const VARIABLE_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;
const DEFAULT_SCHEMA_ID = "shipment";

/** The charge policy as the form holds it: amounts as numbers or strings, bounds null when none. */
export type EditorPolicy = {
  minCharge: number | string | null | undefined;
  maxCharge: number | string | null | undefined;
  roundingMode: FormulaRoundingMode | undefined;
  roundingPrecision: number | undefined;
};

function amountString(value: number | string | null | undefined): string | null {
  if (value === null || value === undefined || value === "") return null;
  const text = String(value).trim();
  return text === "" || Number.isNaN(Number(text)) ? null : text;
}

/** The policy in the shape the server reads, so the assistant sees the minimum, maximum and rounding. */
export function policyDraft(policy: EditorPolicy): FormulaPolicy {
  return {
    minCharge: amountString(policy.minCharge),
    maxCharge: amountString(policy.maxCharge),
    ...(policy.roundingMode ? { roundingMode: policy.roundingMode } : {}),
    ...(policy.roundingPrecision !== undefined
      ? { roundingPrecision: policy.roundingPrecision }
      : {}),
  };
}

/** A variable as the editor's form holds it, before its defaults are filled in. */
export type FormulaEditorVariable = {
  name: string;
  type: VariableValueType;
  description?: string;
  defaultValue?: unknown;
};

export type FormulaEditorState = {
  templateId: string | null;
  schemaId: string;
  templateType: string;
  expression: string;
  variables: readonly FormulaEditorVariable[];
  name?: string;
  description?: string;
  breakdown?: readonly { name?: string; label?: string; expression?: string }[];
  policy?: EditorPolicy;
};

type DraftScalar = FormulaDraftVariable["defaultValue"];

function scalarDefault(value: unknown): DraftScalar {
  if (typeof value === "number") {
    return Number.isFinite(value) ? value : undefined;
  }
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "string") {
    return Array.from(value).length <= MAX_VARIABLE_TEXT_LENGTH ? value : undefined;
  }
  return undefined;
}

/**
 * The server reads three variable types. A Date, list or object variable is
 * still one the formula can reference, so it is sent as text and says what it
 * really is rather than being left out.
 */
function draftVariable(variable: FormulaEditorVariable): FormulaDraftVariable {
  const description = (variable.description ?? "").trim();
  switch (variable.type) {
    case "Number":
    case "String":
    case "Boolean":
      return {
        name: variable.name,
        type: variable.type,
        description: limitCodePoints(description, MAX_VARIABLE_TEXT_LENGTH),
        defaultValue: scalarDefault(variable.defaultValue),
      };
    default: {
      // i18n-ignore: variable description sent to the drafting model in the request payload
      const kind = `A ${variable.type} value.`;
      return {
        name: variable.name,
        type: "String",
        description: limitCodePoints(
          description === "" ? kind : `${kind} ${description}`,
          MAX_VARIABLE_TEXT_LENGTH,
        ),
        defaultValue: undefined,
      };
    }
  }
}

/** What the formula editor holds, in the shape and within the bounds the server reads. */
export function formulaDraftFromEditor(editor: FormulaEditorState): PageDraft {
  const seen = new Set<string>();
  const variables: FormulaDraftVariable[] = [];
  for (const variable of editor.variables) {
    if (variables.length === MAX_VARIABLES) {
      break;
    }
    const name = variable.name.trim();
    if (name.length > MAX_VARIABLE_NAME_LENGTH || !VARIABLE_NAME.test(name) || seen.has(name)) {
      continue;
    }
    seen.add(name);
    variables.push(draftVariable({ ...variable, name }));
  }

  const templateId = editor.templateId?.trim() ?? "";
  const name = limitCodePoints((editor.name ?? "").trim(), MAX_TEMPLATE_NAME_LENGTH);
  const description = limitCodePoints((editor.description ?? "").trim(), MAX_VARIABLE_TEXT_LENGTH);
  const breakdown: FormulaDraftBreakdown[] = [];
  for (const line of editor.breakdown ?? []) {
    if (breakdown.length === MAX_BREAKDOWN_LINES) break;
    const lineName = (line.name ?? "").trim();
    const expression = (line.expression ?? "").trim();
    if (!VARIABLE_NAME.test(lineName) || expression === "") continue;
    breakdown.push({
      name: lineName,
      label: limitCodePoints((line.label ?? "").trim(), MAX_BREAKDOWN_LABEL),
      expression: limitCodePoints(expression, MAX_EXPRESSION_LENGTH),
    });
  }

  return {
    surface: "formula",
    formula: {
      ...(templateId === "" ? {} : { templateId }),
      schemaId: editor.schemaId.trim() || DEFAULT_SCHEMA_ID,
      templateType: editor.templateType.trim(),
      ...(name === "" ? {} : { name }),
      ...(description === "" ? {} : { description }),
      expression: limitCodePoints(editor.expression, MAX_EXPRESSION_LENGTH),
      variables,
      ...(breakdown.length > 0 ? { breakdown } : {}),
      ...(editor.policy ? { policy: policyDraft(editor.policy) } : {}),
    },
  };
}

/** The expression as the editor formats it, or as written when it does not parse. */
export function formattedOrAsWritten(expression: string): string {
  const result = formatExpression(expression);
  return result.ok ? result.text : expression;
}

/** A formula the assistant proposed, as the editor takes it when the person inserts it. */
export function proposalForEditor(proposal: FormulaProposal): {
  expression: string;
  variableDefinitions: VariableDefinition[];
} {
  return {
    expression: formattedOrAsWritten(proposal.expression),
    variableDefinitions: proposal.variables.map((variable) => ({
      name: variable.name,
      type: variable.type,
      description: variable.description,
      required: false,
      defaultValue: variable.defaultValue ?? undefined,
    })),
  };
}

/** The part of the form a proposal can change. */
export type ProposalTarget = {
  name: string;
  description: string;
  type: "FreightCharge" | "AccessorialCharge";
  expression: string;
  variableDefinitions: VariableDefinition[];
  breakdownDefinitions: { name: string; label: string; expression: string }[];
  minCharge: number | null;
  maxCharge: number | null;
  roundingMode: FormulaRoundingMode;
  roundingPrecision: number;
};

function amountNumber(value: string | null | undefined, fallback: number | null): number | null {
  if (value === undefined) return fallback;
  if (value === null || value.trim() === "") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

/** The policy a proposal lands the form on: what it set, over what the editor had. */
export function applyPolicy(
  current: Pick<ProposalTarget, "minCharge" | "maxCharge" | "roundingMode" | "roundingPrecision">,
  policy: FormulaPolicy | null | undefined,
): Pick<ProposalTarget, "minCharge" | "maxCharge" | "roundingMode" | "roundingPrecision"> {
  if (!policy) {
    return {
      minCharge: current.minCharge,
      maxCharge: current.maxCharge,
      roundingMode: current.roundingMode,
      roundingPrecision: current.roundingPrecision,
    };
  }
  return {
    minCharge: amountNumber(policy.minCharge, current.minCharge),
    maxCharge: amountNumber(policy.maxCharge, current.maxCharge),
    roundingMode: policy.roundingMode ?? current.roundingMode,
    roundingPrecision: policy.roundingPrecision ?? current.roundingPrecision,
  };
}

/** Whether a proposal moves the minimum, the maximum or the rounding away from the editor's. */
export function policyChanged(current: ProposalTarget, next: ProposalTarget): boolean {
  return (
    current.minCharge !== next.minCharge ||
    current.maxCharge !== next.maxCharge ||
    current.roundingMode !== next.roundingMode ||
    current.roundingPrecision !== next.roundingPrecision
  );
}

/**
 * What the editor holds once a proposal is inserted: Nova's expression, every
 * variable it needs that the editor lacks (the editor's own are kept as they
 * are), its breakdown lines when it has any, and its name, description and
 * type only where the editor has none of its own.
 */
export function applyProposal(current: ProposalTarget, proposal: FormulaProposal): ProposalTarget {
  const fromProposal = proposalForEditor(proposal);
  const existing = new Set(current.variableDefinitions.map((variable) => variable.name));
  const additions = fromProposal.variableDefinitions.filter(
    (variable) => variable.name !== "" && !existing.has(variable.name),
  );
  const hasName = current.name.trim() !== "";
  const breakdown =
    proposal.breakdown.length > 0
      ? proposal.breakdown.map((line) => ({
          name: line.name,
          label: line.label,
          expression: formattedOrAsWritten(line.expression),
        }))
      : current.breakdownDefinitions;
  const proposed = new Set(fromProposal.variableDefinitions.map((variable) => variable.name));
  const read = identifiersIn([proposal.expression, ...breakdown.map((line) => line.expression)]);
  const kept = current.variableDefinitions.filter(
    (variable) => proposed.has(variable.name) || read.has(variable.name),
  );

  return {
    name: hasName ? current.name : proposal.name.trim(),
    description:
      current.description.trim() !== "" ? current.description : proposal.description.trim(),
    type: hasName || !proposal.templateType ? current.type : proposal.templateType,
    expression: fromProposal.expression,
    variableDefinitions: [...kept, ...additions],
    breakdownDefinitions: breakdown,
    ...applyPolicy(current, proposal.policy),
  };
}

const IDENTIFIER = /[A-Za-z_][A-Za-z0-9_]*/g;

/** Every bare name the expressions mention, so an input the formula stopped reading is let go. */
function identifiersIn(expressions: readonly string[]): Set<string> {
  const names = new Set<string>();
  for (const expression of expressions) {
    for (const match of expression.matchAll(IDENTIFIER)) names.add(match[0]);
  }
  return names;
}

/** The variables a proposal adds that the editor does not have yet. */
export function newProposalVariables(
  current: readonly { name: string }[],
  proposal: FormulaProposal,
): FormulaDraftVariable[] {
  const existing = new Set(current.map((variable) => variable.name));
  return proposal.variables.filter((variable) => !existing.has(variable.name));
}
