import type { PreviewReason, PreviewWarning } from "@/lib/graphql/agent-preview";
import type { ProposalField } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { humanizeToolName } from "../proposal-state";

/**
 * The server names what a person deciding should know with a code and sends
 * an English sentence beside it. The code is what is translated; the English
 * is shown only for a code this client does not know yet, so a warning added
 * on the server is never dropped on the way to the screen.
 */
export const PREVIEW_WARNING_CODES = [
  "would_fail",
  "already_told_customer",
  "driver_unreachable",
  "depends_on_step",
  "target_changed",
  "record_missing",
  "tool_removed",
  "preview_failed",
  "withheld",
  "sensitive_content",
  "retarget_refused",
  "unpinned",
] as const;

export type PreviewWarningCode = (typeof PREVIEW_WARNING_CODES)[number];

/**
 * How loudly a warning is drawn. `danger` says the write will not do what was
 * asked; `warning` says it will, with a consequence to weigh; `info` says how
 * to read the preview itself.
 */
export type PreviewWarningTone = "danger" | "warning" | "info";

const TONES: Record<PreviewWarningCode, PreviewWarningTone> = {
  would_fail: "danger",
  record_missing: "danger",
  tool_removed: "danger",
  retarget_refused: "danger",
  already_told_customer: "warning",
  driver_unreachable: "warning",
  target_changed: "warning",
  preview_failed: "warning",
  sensitive_content: "warning",
  depends_on_step: "info",
  withheld: "info",
  unpinned: "info",
};

export function isPreviewWarningCode(code: string): code is PreviewWarningCode {
  return Object.hasOwn(TONES, code);
}

export function previewWarningTone(warning: PreviewWarning): PreviewWarningTone {
  return isPreviewWarningCode(warning.code) ? TONES[warning.code] : "warning";
}

/** The warning's first argument, when the server sent one worth reading. */
function firstArg(warning: PreviewWarning): string {
  return warning.args.find((arg) => arg.trim() !== "")?.trim() ?? "";
}

/**
 * One rule a write would break: the field the rule names in the record's
 * own words (empty for a refusal of the whole write), what is wrong, and the
 * proposal parameter that carries the field, when the call carries it.
 */
export type WouldFailReason = PreviewReason;

/**
 * The sentences the server leads a refusal with. They are stripped so the
 * person reads the reason itself, never a heading with the reason folded
 * into it.
 */
const REFUSAL_PREFIXES = [
  "This would be refused as it stands:",
  "This change would fail as proposed:",
  "validation failed:",
];

function stripRefusalPrefixes(text: string): string {
  let rest = text.trim();
  let stripped = true;
  while (stripped) {
    stripped = false;
    for (const prefix of REFUSAL_PREFIXES) {
      if (rest.startsWith(prefix)) {
        rest = rest.slice(prefix.length).trim();
        stripped = true;
      }
    }
  }

  return rest;
}

/**
 * What a would_fail warning is made of. The server's structured reasons when
 * it sent them; otherwise its message, without the sentence that leads it,
 * split into the lines a validation failure lists. A server that said why
 * is never reduced to the bare "would not go through".
 */
export function wouldFailReasons(warning: PreviewWarning): WouldFailReason[] {
  if (warning.code !== "would_fail") {
    return [];
  }
  const structured = warning.reasons.filter((entry) => entry.message.trim() !== "");
  if (structured.length > 0) {
    return structured.map((entry) => ({
      field: entry.field,
      label: entry.label,
      message: entry.message.trim(),
      param: entry.param,
    }));
  }

  const text = stripRefusalPrefixes(firstArg(warning) || warning.message);
  if (text === "") {
    return [];
  }

  return text
    .split(/\n-\s*/u)
    .map((line) => stripRefusalPrefixes(line.replace(/^-\s*/u, "")))
    .filter((line) => line !== "")
    .map((message) => ({ field: "", label: "", message, param: "" }));
}

/** A reason in one line: "BOL: already in use", or the message alone. */
export function reasonText(entry: WouldFailReason): string {
  return entry.label !== "" ? `${entry.label}: ${entry.message}` : entry.message;
}

/**
 * Whether a person approving may change the parameter a reason names: its
 * top-level field is one the editor offers and not the record the write is
 * about. A nested parameter (shipment.bol) is edited inside its field.
 */
export function editableParam(param: string, fields: readonly ProposalField[]): boolean {
  const top = param.split(/[.[]/u, 1)[0] ?? "";
  if (top === "") {
    return false;
  }
  const field = fields.find((entry) => entry.name === top);

  return field !== undefined && field.readOnly !== true;
}

/**
 * What the person tells the agent when they want the proposal fixed rather
 * than approved as it stands: each reason, and that a corrected proposal is
 * wanted, so the agent proposes again instead of leaving it.
 */
export function askAgentMessage(
  toolName: string,
  reasons: readonly WouldFailReason[],
  t: TranslateFn,
): string {
  const tool = humanizeToolName(toolName);
  const what = tool.charAt(0).toLowerCase() + tool.slice(1);
  const listed = reasons.map(reasonText).join("; ");

  return listed === ""
    ? t(
        "The proposal to {0} would not go through as it stands. Fix it and propose it again; ask me for anything you need.",
        what,
      )
    : t(
        "The proposal to {0} would not go through as it stands: {1}. Fix it and propose it again; ask me for anything you need.",
        what,
        listed,
      );
}

/** The same request for a plan, which is named by its title rather than a tool. */
export function askAgentPlanMessage(
  title: string,
  reasons: readonly WouldFailReason[],
  t: TranslateFn,
): string {
  const listed = reasons.map(reasonText).join("; ");

  return listed === ""
    ? t(
        'The plan "{0}" would not go through as it stands. Fix it and propose it again; ask me for anything you need.',
        title,
      )
    : t(
        'The plan "{0}" would not go through as it stands: {1}. Fix it and propose it again; ask me for anything you need.',
        title,
        listed,
      );
}

export function previewWarningText(warning: PreviewWarning, t: TranslateFn): string {
  const arg = firstArg(warning);

  switch (warning.code) {
    case "would_fail": {
      const listed = wouldFailReasons(warning).map(reasonText).join("; ");

      return listed !== ""
        ? t("This would not go through as it stands: {0}", listed)
        : t("This would not go through as it stands.");
    }
    case "already_told_customer":
      return t("The customer has already been told about this.");
    case "driver_unreachable":
      return arg !== ""
        ? t("{0} has no driver portal access, so the message would not reach them.", arg)
        : t("The driver has no driver portal access, so the message would not reach them.");
    case "depends_on_step":
      return arg !== ""
        ? t("Step {0} changes this record first; it is shown as it would be after that step.", arg)
        : t("An earlier step changes this record first; it is shown as it would be after it.");
    case "target_changed":
      return t("The record has been edited since this was proposed.");
    case "record_missing":
      return t("The record this would change is no longer there.");
    case "tool_removed":
      return t("The action this would run is no longer available.");
    case "preview_failed":
      return t(
        "What this would change could not be worked out; the values it would run with are shown instead.",
      );
    case "withheld":
      return t("Some of what this changes is hidden by your data access.");
    case "sensitive_content":
      return t("The message contains details your organization marks as sensitive.");
    case "retarget_refused":
      return t(
        "A change can't be pointed at a different record; ask the agent for a new proposal instead.",
      );
    case "unpinned":
      return t(
        "This was proposed without noting the record's version, so later edits to it can't be detected.",
      );
    default:
      return warning.message;
  }
}

/**
 * The warnings a surface shows. A plan draws "uses the record step 2 changes"
 * beside the step itself, so it drops the warning that says the same.
 */
export function visibleWarnings(
  warnings: readonly PreviewWarning[],
  { inPlan }: { inPlan: boolean },
): PreviewWarning[] {
  return inPlan ? warnings.filter((warning) => warning.code !== "depends_on_step") : [...warnings];
}
