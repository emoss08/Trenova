import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * What made an agent look back over its work, in a person's words. A memory's
 * evidence carries the same kinds as plain strings, so an unknown one is
 * shown as it was recorded rather than dropped.
 */
export function reflectionSignalLabel(kind: string, t: TranslateFn): string {
  switch (kind) {
    case "ToolRecovered":
      return t("A tool worked after failing");
    case "ToolFailed":
      return t("A tool failed");
    case "PersonCorrected":
      return t("A person corrected it");
    case "StandingRequest":
      return t("A person said how they want it done");
    case "ProposalModified":
      return t("A proposal was changed");
    case "ProposalRejected":
      return t("A proposal was refused");
    case "NegativeFeedback":
      return t("A reply was rated unhelpful");
    case "LongTask":
      return t("A long task");
    default:
      return kind;
  }
}
