import type { JournalReviewResult } from "@/lib/graphql/journal-review";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export type JournalReviewAction = "approve" | "post";

export type JournalReviewMessage = {
  tone: "success" | "warning" | "error";
  title: string;
  description?: string;
};

export function journalReviewMessage(
  t: TranslateFn,
  action: JournalReviewAction,
  result: JournalReviewResult,
): JournalReviewMessage {
  const changed =
    action === "approve"
      ? t("{0} approved", result.changed)
      : t("{0} posted to the general ledger", result.changed);
  const refused = result.outcomes.filter((outcome) => outcome.error !== "");
  if (refused.length === 0) {
    return { tone: "success", title: changed };
  }

  const first = refused[0];
  const detail = first.entryNumber ? `${first.entryNumber}: ${first.error}` : first.error;
  const description =
    refused.length > 1 ? t("{0} (and {1} more)", detail, refused.length - 1) : detail;
  return {
    tone: result.changed > 0 ? "warning" : "error",
    title:
      result.changed > 0
        ? t("{0}; {1} could not be changed", changed, result.failed)
        : t("None of the {0} selected could be changed", result.failed),
    description,
  };
}
