import type { FormulaDraftVariable, FormulaProposal, PageDraft } from "@/types/page-draft";
import { limitCodePoints } from "@trenova/shared/lib/utils";
import type { VariableDefinition, VariableValueType } from "@trenova/shared/types/formula-template";

/* The bounds pagedraft.Formula enforces; past any of them the turn is refused. */
const MAX_EXPRESSION_LENGTH = 10_000;
const MAX_VARIABLES = 50;
const MAX_VARIABLE_NAME_LENGTH = 64;
const MAX_VARIABLE_TEXT_LENGTH = 500;
const VARIABLE_NAME = /^[A-Za-z_][A-Za-z0-9_]*$/;
const DEFAULT_SCHEMA_ID = "shipment";

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

  return {
    surface: "formula",
    formula: {
      ...(templateId === "" ? {} : { templateId }),
      schemaId: editor.schemaId.trim() || DEFAULT_SCHEMA_ID,
      templateType: editor.templateType.trim(),
      expression: limitCodePoints(editor.expression, MAX_EXPRESSION_LENGTH),
      variables,
    },
  };
}

/** A formula the assistant proposed, as the editor takes it when the person inserts it. */
export function proposalForEditor(proposal: FormulaProposal): {
  expression: string;
  variableDefinitions: VariableDefinition[];
} {
  return {
    expression: proposal.expression,
    variableDefinitions: proposal.variables.map((variable) => ({
      name: variable.name,
      type: variable.type,
      description: variable.description,
      required: false,
      defaultValue: variable.defaultValue ?? undefined,
    })),
  };
}
