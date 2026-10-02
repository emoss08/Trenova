import { useT } from "@trenova/shared/i18n/use-t";
import { DOCUMENT_REVIEW_PHASE, documentReviewState } from "@/lib/document-review";
import { Badge } from "@trenova/shared/components/ui/badge";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";

type DocumentReviewBadgeProps = {
  document: Pick<Document, "status" | "approvedAt" | "isCurrentVersion" | "rejectionReason">;
  className?: string;
};

export function DocumentReviewBadge({ document, className }: DocumentReviewBadgeProps) {
  const t = useT();
  const state = documentReviewState(document);

  if (state === "unreviewed" || state === "inactive") {
    return null;
  }

  const label =
    state === "approved" ? t("Approved") : state === "rejected" ? t("Rejected") : t("Awaiting review");
  const description =
    state === "rejected" && document.rejectionReason
      ? t("Rejected: {0}", document.rejectionReason)
      : undefined;

  return (
    <Badge
      variant={phaseTone(DOCUMENT_REVIEW_PHASE[state])}
      title={description}
      className={cn("h-5 px-1.5 py-0 text-2xs", className)}
    >
      {label}
    </Badge>
  );
}
