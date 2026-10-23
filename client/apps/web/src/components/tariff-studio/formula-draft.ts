import type { SourceProposal, PageDraft } from "@/types/page-draft";
import { limitCodePoints, limitUtf8Bytes } from "@trenova/shared/lib/utils";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffFormat } from "@trenova/shared/rules/tariff";

/* The bounds pagedraft.Formula and pagedraft.FormulaLibrary enforce; past any of them the turn is
 * refused. The source bound is the module's own limit on a file, in UTF-8 bytes. */
export const MAX_SOURCE_BYTES = 64 * 1024;
const MAX_TEMPLATE_NAME_LENGTH = 100;
const MAX_LIBRARY_NAME_LENGTH = 64;
const MAX_DESCRIPTION_LENGTH = 2_000;

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
  const name = limitCodePoints(
    (editor.name ?? "").trim(),
    editor.kind === "library" ? MAX_LIBRARY_NAME_LENGTH : MAX_TEMPLATE_NAME_LENGTH,
  );
  const description = limitCodePoints((editor.description ?? "").trim(), MAX_DESCRIPTION_LENGTH);
  const file = {
    ...(name === "" ? {} : { name }),
    ...(description === "" ? {} : { description }),
    source: limitUtf8Bytes(editor.source, MAX_SOURCE_BYTES),
  };
  if (editor.kind === "library") {
    return {
      surface: "formula_library",
      formulaLibrary: { ...(recordId === "" ? {} : { libraryId: recordId }), ...file },
    };
  }
  return {
    surface: "formula",
    formula: { ...(recordId === "" ? {} : { templateId: recordId }), ...file },
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
  proposal: SourceProposal,
): ProposalTarget {
  return {
    name: current.name.trim() !== "" ? current.name : proposal.name.trim(),
    description:
      current.description.trim() !== "" ? current.description : proposal.description.trim(),
    source: formattedOrAsWritten(rules, proposal.source),
  };
}
