import { RatingReviewHold } from "@/components/billing/rating-review-hold";
import { useQueryClient } from "@tanstack/react-query";
import type { RatingReviewHold as RatingReviewHoldValue } from "@trenova/shared/types/billing-queue";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

/**
 * The reviews the tariff asked for on the item's shipment. The item cannot be approved until
 * the rating is; approving it here refreshes the item so its checks and approve action follow.
 */
export function BillingQueueRatingReviewHold({ hold }: { hold: RatingReviewHoldValue | null }) {
  const invalidate = useInvalidateBillingQueue();
  const queryClient = useQueryClient();

  if (!hold || hold.reasons.length === 0) {
    return null;
  }

  return (
    <div className="shrink-0 px-4 pt-2" data-testid="billing-queue-rating-review-hold">
      <RatingReviewHold
        shipmentId={hold.shipmentId}
        reasons={hold.reasons}
        awaiting
        onApproved={() => {
          invalidate();
          void queryClient.invalidateQueries({ queryKey: ["shipment"] });
        }}
      />
    </div>
  );
}
