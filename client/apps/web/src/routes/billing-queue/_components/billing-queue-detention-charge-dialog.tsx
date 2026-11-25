import { WaiveDialog } from "@/components/detention/occurrence-action-dialogs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { detention } from "@/lib/queries/detention";
import { detentionService } from "@/services/detention";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { BillingQueueDetentionChargeSkeleton } from "./billing-queue-detention-charge-skeleton";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

function standing(t: TranslateFn, occurrence: DetentionOccurrence): string {
  switch (occurrence.status) {
    case "Accruing":
      return t("Still at the dock");
    case "Pending":
      return occurrence.requiresApproval ? t("Waiting on approval") : t("Will bill");
    case "Approved":
      return t("Approved");
    case "Billed":
      return t("Billed");
    case "Waived":
      return t("Waived");
    case "Disputed":
      return t("Disputed by the customer");
    case "NotBillable":
      return t("Not billable");
  }
}

/** A charge can still be approved while it waits on a person, and waived until it is billed. */
function canApprove(occurrence: DetentionOccurrence): boolean {
  return occurrence.status === "Pending";
}

function canWaive(occurrence: DetentionOccurrence): boolean {
  return occurrence.status === "Pending" || occurrence.status === "Approved";
}

/**
 * The stops behind a shipment's detention line, each with where it stands and what it bills.
 * A biller decides each where it is listed: approving bills it, and waiving takes it off the
 * shipment's detention line and the invoice this item will make.
 */
export function BillingQueueDetentionChargeDialog({
  shipmentId,
  open,
  onOpenChange,
}: {
  shipmentId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const { allowed: canDecide } = usePermission(Resource.Detention, Operation.Update);
  const { data: occurrences, isPending } = useQuery({
    ...detention.byShipment(shipmentId),
    enabled: open,
  });
  const charges = (occurrences ?? []).filter(
    (occurrence) => occurrence.billableAmount > 0 || occurrence.waivedAmount > 0,
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Detention on this shipment")}</DialogTitle>
          <DialogDescription>
            {canDecide
              ? t(
                  "Each detained stop bills on this one line. Approve a charge to bill it, or waive it to take it off the line and the invoice.",
                )
              : t("Each detained stop bills on this one line, as listed here.")}
          </DialogDescription>
        </DialogHeader>
        {isPending ? (
          <BillingQueueDetentionChargeSkeleton />
        ) : charges.length === 0 ? (
          <p className="text-foreground-muted text-sm">
            {t("No stop on this shipment bills detention.")}
          </p>
        ) : (
          <ul className="flex flex-col">
            {charges.map((occurrence) => (
              <ChargeRow key={occurrence.id} occurrence={occurrence} canDecide={canDecide} />
            ))}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  );
}

function ChargeRow({
  occurrence,
  canDecide,
}: {
  occurrence: DetentionOccurrence;
  canDecide: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidateQueue = useInvalidateBillingQueue();
  const [waiveOpen, setWaiveOpen] = useState(false);

  const refresh = () => {
    invalidateQueue();
    void queryClient.invalidateQueries({ queryKey: detention._def });
  };

  const approve = useApiMutation<DetentionOccurrence, undefined>({
    scope: { id: `detention-decide:${occurrence.id}` },
    mutationFn: () => detentionService.approve(occurrence.id),
    onSuccess: refresh,
    resourceName: "Detention Occurrence",
  });

  const waived = occurrence.status === "Waived";
  const amount = waived ? occurrence.waivedAmount : occurrence.billableAmount;

  return (
    <li
      data-testid="billing-queue-detention-charge"
      className="border-border flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-b py-2.5 last:border-b-0"
    >
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="truncate font-medium">
          {occurrence.locationName || t("Detention charge")}
        </span>
        <span className="text-foreground-muted text-xs">{standing(t, occurrence)}</span>
      </span>
      <span className="flex items-center gap-2">
        <span
          className={waived ? "text-foreground-muted tabular-nums line-through" : "tabular-nums"}
        >
          {formatCurrency(amount, occurrence.currency)}
        </span>
        {canDecide && canApprove(occurrence) ? (
          <Button
            size="xs"
            variant="outline"
            isLoading={approve.isPending}
            loadingText={t("Approving")}
            onClick={() => void approve.mutateAsync(undefined).catch(() => undefined)}
          >
            {t("Approve")}
          </Button>
        ) : null}
        {canDecide && canWaive(occurrence) ? (
          <Button
            size="xs"
            variant="outline"
            disabled={approve.isPending}
            onClick={() => setWaiveOpen(true)}
          >
            {t("Waive")}
          </Button>
        ) : null}
      </span>
      {canDecide && canWaive(occurrence) ? (
        <WaiveDialog
          occurrenceId={occurrence.id}
          open={waiveOpen}
          onOpenChange={setWaiveOpen}
          onDone={refresh}
        />
      ) : null}
    </li>
  );
}
