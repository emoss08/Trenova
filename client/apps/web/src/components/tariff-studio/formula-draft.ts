import type { FormulaProposal, PageDraft } from "@/types/page-draft";
import { limitCodePoints } from "@trenova/shared/lib/utils";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffFormat } from "@trenova/shared/rules/tariff";

/* The bounds pagedraft.Formula enforces; past any of them the turn is refused. The source bound
 * is the module's own limit on a file. */
export const MAX_SOURCE_LENGTH = 256 * 1024;
const MAX_NAME_LENGTH = 100;
const MAX_DESCRIPTION_LENGTH = 500;

/** The Tariff file the Studio holds: a template's or a library's, saved or not. */
export type FormulaEditorState = {
  kind: "tariff" | "library";
  /** The saved template's or library's ID; absent while it is being created. */
  recordId?: string;
  name?: string;
  description?: string;
  source: string;
};

/** What the Studio holds, in the shape and within the bounds the server reads. */
export function formulaDraftFromEditor(editor: FormulaEditorState): PageDraft {
  const recordId = editor.recordId?.trim() ?? "";
  const name = limitCodePoints((editor.name ?? "").trim(), MAX_NAME_LENGTH);
  const description = limitCodePoints((editor.description ?? "").trim(), MAX_DESCRIPTION_LENGTH);
  const id =
    recordId === ""
      ? {}
      : editor.kind === "library"
        ? { libraryId: recordId }
        : { templateId: recordId };
  return {
    surface: editor.kind === "library" ? "formula_library" : "formula",
    formula: {
      ...id,
      kind: editor.kind,
      ...(name === "" ? {} : { name }),
      ...(description === "" ? {} : { description }),
      source: limitCodePoints(editor.source, MAX_SOURCE_LENGTH),
    },
  };
}

/** The source in the module's one canonical style, or as written when it has syntax errors. */
export function formattedOrAsWritten(rules: RulesRuntime, source: string): string {
  return tariffFormat(rules, source).formatted ?? source;
}

/** The part of the Studio's form a proposal can change. */
export type ProposalTarget = { name: string; description: string; source: string };

/**
 * What the Studio holds once a proposal is applied: Nova's source, formatted, and its name and
 * description only where the Studio has none of its own.
 */
export function applyProposal(
  rules: RulesRuntime,
  current: ProposalTarget,
  proposal: FormulaProposal,
): ProposalTarget {
  return {
    name: current.name.trim() !== "" ? current.name : proposal.name.trim(),
    description:
      current.description.trim() !== "" ? current.description : proposal.description.trim(),
    source: formattedOrAsWritten(rules, proposal.source),
  };
}
