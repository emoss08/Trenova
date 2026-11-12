import type { PreviewReason, PreviewWarning, ProposalPreview } from "@/lib/graphql/agent-preview";
import type { ProposalField } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { humanizeToolName, isRecordId } from "../proposal-state";

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
  "rate_coverage",
  "recipient_skipped",
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
  rate_coverage: "warning",
  recipient_skipped: "warning",
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

/**
 * The warning's first argument, when the server sent one worth reading. A
 * record id is never one: a warning names a person or a record by what it is
 * called, and says "the driver" when it has no name to give.
 */
function firstArg(warning: PreviewWarning): string {
  return warning.args.find((arg) => arg.trim() !== "" && !isRecordId(arg.trim()))?.trim() ?? "";
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

const MISSING_PARAMETER = /^missing required parameter\s+["'`]?([A-Za-z0-9_.[\]]+)["'`]?\.?$/iu;
const RECORD_ID = /\b[a-z]{2,10}_[0-9A-HJKMNP-TV-Z]{26}\b/gu;

/** A parameter's name as a person would say it: `credentialIds` is "credentials". */
function paramWords(param: string): string {
  const last = (param.split(".").at(-1) ?? param).replace(/\[\d+\]$/u, "");
  const many = /(?:Ids|IDs|_ids)$/u.test(last);
  const named = last.replace(/(?:Ids|IDs|Id|ID|_ids|_id)$/u, "");
  const words = named
    .replace(/_/gu, " ")
    .replace(/([a-z0-9])([A-Z])/gu, "$1 $2")
    .trim()
    .toLowerCase();
  if (words === "") {
    return "";
  }

  return many && !words.endsWith("s") ? `${words}s` : words;
}

/**
 * What is wrong, as the person deciding reads it; the field it is about is
 * drawn beside it. A tool's own refusal is written
 * for the agent that called it ("missing required parameter \"credentialIds\"")
 * and may quote a record's id; here it says what is missing in plain words,
 * and no id is ever shown.
 */
export function reasonSentence(entry: WouldFailReason, t: TranslateFn): string {
  const missing = MISSING_PARAMETER.exec(entry.message);
  if (missing) {
    const words = entry.label !== "" ? entry.label : paramWords(missing[1] ?? "");

    return words !== ""
      ? t("It was proposed without the {0} it needs.", words)
      : t("It was proposed without something it needs.");
  }
  const message = entry.message
    .replace(RECORD_ID, "")
    .replace(/\s{2,}/gu, " ")
    .replace(/\s+([.,;:])/gu, "$1")
    .trim();

  return message;
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
    case "rate_coverage":
      return arg !== ""
        ? t("The rate needs review: {0}", arg)
        : t("Nothing prices this shipment, so it would be saved at zero and flagged for review.");
    case "recipient_skipped":
      return arg !== ""
        ? t("{0} is on the suppression list and would be skipped.", warning.args.join(", "))
        : t("A recipient is on the suppression list and would be skipped.");
    default:
      return warning.message;
  }
}

/**
 * The warnings a surface shows: the ones a person deciding can weigh or act
 * on, each said once. A plan draws "uses the record step 2 changes" beside
 * the step itself, so it drops the warning that says the same.
 */
export function visibleWarnings(
  warnings: readonly PreviewWarning[],
  { inPlan }: { inPlan: boolean },
): PreviewWarning[] {
  return warnings.filter((warning) => {
    switch (warning.code) {
      case "depends_on_step":
        return !inPlan;
      // Nothing a person deciding can do with either: how the record was
      // pinned is the server's bookkeeping, and what their access hides is
      // counted under the change itself.
      case "unpinned":
      case "withheld":
        return false;
      // A preview that could not be worked out shows the values the change
      // would run with under a caption that says so; a note saying it again
      // is one fact twice.
      case "preview_failed":
        return false;
      default:
        return true;
    }
  });
}

/**
 * Whether a preview carries something its decider must see before answering:
 * the record moved since the proposal, or a warning the surface shows.
 */
export function previewNeedsAttention(
  preview: Pick<ProposalPreview, "stale" | "warnings">,
  { inPlan }: { inPlan: boolean },
): boolean {
  return preview.stale || visibleWarnings(preview.warnings, { inPlan }).length > 0;
}
