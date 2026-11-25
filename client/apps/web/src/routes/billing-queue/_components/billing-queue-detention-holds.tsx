import { Link } from "@trenova/shared/components/link";
import { recordPath } from "@/config/record-links";
import { WaiveDialog } from "@/components/detention/occurrence-action-dialogs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { detention } from "@/lib/queries/detention";
import { detentionService } from "@/services/detention";
import { useQueryClient } from "@tanstack/react-query";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { DetentionHold } from "@trenova/shared/types/billing-queue";
import type { BillingHoldReason, DetentionOccurrence } from "@trenova/shared/types/detention";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ClockStopwatchIcon } from "@trenova/shared/components/icons";
import { LockedReference } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { useState } from "react";
import { BillingQueueNotice } from "./billing-queue-notice";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

function holdReasonLabel(t: TranslateFn, reason: BillingHoldReason): string {
  switch (reason) {
    case "OverApprovalThreshold":
      return t("Over the approval threshold");
    case "NoticeNotSent":
      return t("Notice not sent in time");
    case "Escalated":
      return t("Escalated for review");
  }
}

/**
 * The detention charges keeping this item from being approved, each decided where it is
 * listed: approving bills it, waiving takes it off the shipment and the invoice. Without
 * the right to decide detention, each still links to the charge on the detention desk.
 */
export function BillingQueueDetentionHolds({ holds }: { holds: DetentionHold[] }) {
  const t = useT();
  const canReadDetention = useCanReadKind("detention");
  const { allowed: canDecide } = usePermission(Resource.Detention, Operation.Update);

  if (holds.length === 0) {
    return null;
  }

  return (
    <BillingQueueNotice data-testid="billing-queue-detention-holds">
      <ClockStopwatchIcon />
      <AlertTitle>
        {t(
          "{0, plural, one {# detention charge needs approval} other {# detention charges need approval}}",
          holds.length,
        )}
      </AlertTitle>
      <AlertDescription>
        <p>
          {canDecide
            ? t(
                "This item cannot be approved until each charge is approved or waived. A waived charge comes off the shipment and the invoice.",
              )
            : t(
                "This item cannot be approved until each charge is approved or waived by someone who decides detention.",
              )}
        </p>
        <ul className="grid w-full gap-1.5">
          {holds.map((hold) => (
            <HoldRow
              key={hold.occurrenceId}
              hold={hold}
              canReadDetention={canReadDetention}
              canDecide={canDecide}
            />
          ))}
        </ul>
      </AlertDescription>
    </BillingQueueNotice>
  );
}

function HoldRow({
  hold,
  canReadDetention,
  canDecide,
}: {
  hold: DetentionHold;
  canReadDetention: boolean;
  canDecide: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidateQueue = useInvalidateBillingQueue();
  const [waiveOpen, setWaiveOpen] = useState(false);
  const name = hold.locationName || t("Detention charge");

  const refresh = () => {
    invalidateQueue();
    void queryClient.invalidateQueries({ queryKey: detention._def });
  };

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    scope: { id: `detention-decide:${hold.occurrenceId}` },
    mutationFn: () => detentionService.approve(hold.occurrenceId),
    onSuccess: refresh,
    resourceName: "Detention Occurrence",
  });

  return (
    <li
      className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1"
      data-testid="billing-queue-detention-hold"
    >
      {canReadDetention ? (
        <Link
          to={recordPath("detention_occurrence", hold.occurrenceId)}
          className="ui-focus-ring text-brand rounded-control underline-offset-2 hover:underline"
        >
          {name}
        </Link>
      ) : (
        <LockedReference kind="detention">{name}</LockedReference>
      )}
      <span className="flex flex-wrap items-center gap-2">
        <span className="tabular-nums">
          {formatCurrency(hold.billableAmount ?? 0, hold.currency)}
        </span>
        <span>{holdReasonLabel(t, hold.reason)}</span>
        {canDecide ? (
          <>
            <Button
              size="xs"
              variant="outline"
              isLoading={approve.isPending}
              loadingText={t("Approving")}
              onClick={() => void approve.mutateAsync(undefined).catch(() => undefined)}
            >
              {t("Approve")}
            </Button>
            <Button
              size="xs"
              variant="outline"
              disabled={approve.isPending}
              onClick={() => setWaiveOpen(true)}
            >
              {t("Waive")}
            </Button>
          </>
        ) : null}
      </span>
      {canDecide ? (
        <WaiveDialog
          occurrenceId={hold.occurrenceId}
          open={waiveOpen}
          onOpenChange={setWaiveOpen}
          onDone={refresh}
        />
      ) : null}
    </li>
  );
}
