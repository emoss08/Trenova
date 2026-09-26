import type { PreviewWarning } from "@/lib/graphql/agent-preview";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

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

export function previewWarningText(warning: PreviewWarning, t: TranslateFn): string {
  const arg = firstArg(warning);

  switch (warning.code) {
    case "would_fail":
      return arg !== ""
        ? t("This would not go through as it stands: {0}", arg)
        : t("This would not go through as it stands.");
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
