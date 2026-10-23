import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  assignBillingQueueBillerGraphQL,
  updateBillingQueueStatusGraphQL,
} from "@/lib/graphql/billing-queue";
import { shipment } from "@/lib/queries/shipment";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  BillingQueueItem,
  BillingQueueUpdateStatusInput,
} from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  FlipBackwardIcon,
  PauseIcon,
  PlayIcon,
  Send01Icon,
  UserPlus01Icon,
} from "@trenova/shared/components/icons";
import { BillingQueueExceptionPopover } from "./billing-queue-exception-popover";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

export function BillingQueueActionBar({
  item,
  onAssignBiller,
  onAutoAdvance,
}: {
  item: BillingQueueItem;
  onAssignBiller: () => void;
  onAutoAdvance?: () => void;
}) {
  const t = useT();

  const currentUser = useAuthStore((s) => s.user);
  const invalidate = useInvalidateBillingQueue();

  const { data: billingReadiness } = useQuery({
    ...shipment.billingReadiness(item.shipmentId ?? ""),
    enabled: item.status === "InReview" && !!item.shipmentId,
  });

  const heldCount = item.detentionHolds.length;
  const canApprove =
    billingReadiness?.canMarkReadyToInvoice !== false && heldCount === 0 && !item.ratingReviewHold;
  const missingCount = billingReadiness?.missingRequirements?.length ?? 0;

  const { mutate: updateStatus, isPending: isStatusPending } = useApiMutation({
    mutationFn: (input: BillingQueueUpdateStatusInput) =>
      updateBillingQueueStatusGraphQL(item.id, input),
    resourceName: "BillingQueueItem",
    onSuccess: (_, input) => {
      invalidate();
      if (input.status === "Approved" && onAutoAdvance) {
        onAutoAdvance();
      }
    },
  });

  const { mutate: assignAndReview, isPending: isAssignPending } = useApiMutation({
    mutationFn: (billerId: string) => assignBillingQueueBillerGraphQL(item.id, { billerId }),
    resourceName: "BillingQueueItem",
    onSuccess: () => {
      invalidate();
    },
  });

  const isPending = isStatusPending || isAssignPending;

  switch (item.status) {
    case "ReadyForReview":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => currentUser?.id && assignAndReview(currentUser.id)}
            disabled={isPending || !currentUser?.id}
          >
            <PlayIcon className="size-3.5" />
            {t("Start review")}
          </Button>
          <Button size="sm" variant="outline" onClick={onAssignBiller} disabled={isPending}>
            <UserPlus01Icon className="size-3.5" />
            {t("Assign biller")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => updateStatus({ status: "OnHold" })}
            disabled={isPending}
          >
            <PauseIcon className="size-3.5" />
            {t("Hold")}
          </Button>
        </div>
      );

    case "InReview":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="sm"
                  className="bg-success text-foreground-on-solid hover:bg-success disabled:opacity-50"
                  onClick={() => updateStatus({ status: "Approved" })}
                  disabled={isPending || !canApprove}
                >
                  <CheckIcon className="size-3.5" />
                  {t("Approve")}
                </Button>
              }
            />
            {!canApprove && (
              <TooltipContent side="bottom" sideOffset={8}>
                {heldCount > 0
                  ? t(
                      "{0, plural, one {# detention charge needs approval} other {# detention charges need approval}} before this can be approved",
                      heldCount,
                    )
                  : item.ratingReviewHold
                    ? t("The tariff held this rating for review; approve the rating first")
                    : missingCount > 0
                      ? t(
                          "{0, plural, one {# required document} other {# required documents}} missing",
                          missingCount,
                        )
                      : t("Billing requirements not met")}
              </TooltipContent>
            )}
          </Tooltip>
          <BillingQueueExceptionPopover
            itemId={item.id}
            targetStatus="Exception"
            label={t("Exception")}
            icon={<AlertTriangleIcon className="size-3.5" />}
            variant="destructive"
            disabled={isPending}
            onSuccess={invalidate}
          />
          <BillingQueueExceptionPopover
            itemId={item.id}
            targetStatus="SentBackToOps"
            label={t("Send back")}
            icon={<Send01Icon className="size-3.5" />}
            variant="outline"
            disabled={isPending}
            onSuccess={invalidate}
          />
          <Button
            size="sm"
            variant="outline"
            onClick={() => updateStatus({ status: "OnHold" })}
            disabled={isPending}
          >
            <PauseIcon className="size-3.5" />
            {t("Hold")}
          </Button>
        </div>
      );

    case "OnHold":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <PlayIcon className="size-3.5" />
            {t("Resume")}
          </Button>
        </div>
      );

    case "SentBackToOps":
    case "Exception":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Button
            size="sm"
            onClick={() => updateStatus({ status: "ReadyForReview" })}
            disabled={isPending}
          >
            <FlipBackwardIcon className="size-3.5" />
            {t("Resolve")}
          </Button>
        </div>
      );

    case "Approved":
      return (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => updateStatus({ status: "InReview" })}
                  disabled={isPending}
                >
                  <FlipBackwardIcon className="size-3.5" />
                  {t("Revert to review")}
                </Button>
              }
            />
            <TooltipContent side="bottom" sideOffset={8}>
              {t(
                "The draft invoice is voided and a new one is built from the corrected charges when you approve again",
              )}
            </TooltipContent>
          </Tooltip>
        </div>
      );

    default:
      return null;
  }
}
