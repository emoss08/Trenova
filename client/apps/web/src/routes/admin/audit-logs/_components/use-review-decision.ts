import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  markAuditReviewReviewed,
  reopenAuditReview,
  type AuditReview,
} from "@/lib/graphql/audit-overview";
import { audit } from "@/lib/queries/audit";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Marks a review done or opens it again, for the card and the inspector alike. A review takes one
 * decision at a time, so a second click while the first is in flight sends nothing; every cache
 * that shows reviews reads again when it lands.
 */
export function useReviewDecision(review: Pick<AuditReview, "id">) {
  const queryClient = useQueryClient();
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: audit.overview._def });
    void queryClient.invalidateQueries({ queryKey: audit.reviews._def });
    void queryClient.invalidateQueries({ queryKey: audit.entryReview._def });
  };

  const mark = useApiMutation({
    scope: { id: `audit-review:${review.id}` },
    resourceName: "Audit review",
    mutationFn: () => markAuditReviewReviewed(review.id),
    onSettled: refresh,
  });
  const reopen = useApiMutation({
    scope: { id: `audit-review:${review.id}` },
    resourceName: "Audit review",
    mutationFn: () => reopenAuditReview(review.id),
    onSettled: refresh,
  });

  return {
    markReviewed: () =>
      mark.isPending ? Promise.resolve() : mark.mutateAsync().then(() => undefined),
    reopen: () =>
      reopen.isPending ? Promise.resolve() : reopen.mutateAsync().then(() => undefined),
    marking: mark.isPending,
    reopening: reopen.isPending,
  };
}
