import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { approveShipmentRatingReviewGraphQL } from "@/lib/graphql/shipment";
import { CheckVerified01Icon, Flag01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";

export type RatingReviewHoldProps = {
  shipmentId: string;
  /** The reviews the tariff asked for, in its author's words. */
  reasons: string[];
  /** True while the reviews wait on a person; false once someone has approved them. */
  awaiting: boolean;
  /** Who approved the reviews and when, shown once they are approved. */
  approvedAt?: number | null;
  approvedByName?: string | null;
  onApproved?: (shipment: Shipment) => void;
};

/**
 * The reviews a tariff asked for on a shipment's rating. While they wait, the shipment is rated
 * but held out of billing, and someone who works the billing queue approves the rating here;
 * once approved, it says who approved it and when.
 */
export function RatingReviewHold({
  shipmentId,
  reasons,
  awaiting,
  approvedAt,
  approvedByName,
  onApproved,
}: RatingReviewHoldProps) {
  const t = useT();
  const { allowed: canApprove } = usePermission(Resource.BillingQueue, Operation.Update);

  const { mutate: approve, isPending } = useApiMutation({
    mutationFn: () => approveShipmentRatingReviewGraphQL(shipmentId),
    resourceName: "Shipment",
    onSuccess: (shipment) => onApproved?.(shipment),
  });

  if (reasons.length === 0) {
    return null;
  }

  const list = (
    <ul aria-label={t("Reviews")} className="grid w-full list-disc gap-0.5 pl-4">
      {reasons.map((reason, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- the tariff's reviews have no identity but their order
        <li key={index}>{reason}</li>
      ))}
    </ul>
  );

  if (!awaiting) {
    return (
      <Alert size="sm" variant="success" data-testid="rating-review-approved">
        <CheckVerified01Icon />
        <AlertTitle>{t("Rating reviewed")}</AlertTitle>
        <AlertDescription>
          <p>
            {approvedAt
              ? t(
                  "Approved for billing by {0} on {1}.",
                  approvedByName || t("a teammate"),
                  formatUnixDateMedium(approvedAt),
                )
              : t("Approved for billing.")}
          </p>
          {list}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert size="sm" variant="warning" data-testid="rating-review-hold">
      <Flag01Icon />
      <AlertTitle>
        {t(
          "{0, plural, one {The tariff asks for # review before this shipment bills} other {The tariff asks for # reviews before this shipment bills}}",
          reasons.length,
        )}
      </AlertTitle>
      <AlertDescription>
        <p>
          {t(
            "The charges are applied, but the shipment cannot be billed until someone approves the rating.",
          )}
        </p>
        {list}
        {canApprove ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="mt-1"
            isLoading={isPending}
            loadingText={t("Approving…")}
            onClick={() => approve()}
          >
            {t("Approve rating")}
          </Button>
        ) : (
          <p>{t("Someone who works the billing queue approves it.")}</p>
        )}
      </AlertDescription>
    </Alert>
  );
}
