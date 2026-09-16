"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChargeSplitEditor } from "@/components/billing/charge-split-editor";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  allocationsForCharge,
  nextItemAfterReassignment,
  reassignSummary,
} from "@/lib/billing-queue-charges";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingQueueItem,
  PayerShareLine,
  ReassignChargeInput,
  ReassignChargeResult,
} from "@trenova/shared/types/billing-queue";
import {
  chargeAllocationSchema,
  chargeAllocationsRefinement,
  type ChargeAllocation,
} from "@trenova/shared/types/shipment";
import { useQueryStates } from "nuqs";
import { useEffect, useMemo } from "react";
import {
  FormProvider,
  useForm,
  type Control,
  type FieldValues,
  type Resolver,
} from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { queueSelectionSearchParamsParser } from "../use-billing-queue-state";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

type ReassignFormValues = { allocations: ChargeAllocation[] };

function reassignFormSchema(chargeTotal: number | null) {
  return z
    .object({ allocations: z.array(chargeAllocationSchema) })
    .superRefine((values, ctx) =>
      chargeAllocationsRefinement(ctx, values.allocations, chargeTotal, ["allocations"]),
    );
}

function toReassignInput(line: PayerShareLine, rows: ChargeAllocation[]): ReassignChargeInput {
  return {
    chargeKind: line.kind === "Freight" ? "Freight" : "Accessorial",
    additionalChargeId:
      line.kind === "Freight" ? undefined : (line.additionalChargeId ?? undefined),
    allocations: rows
      .filter((row) => row.billToCustomerId)
      .map((row) => ({
        ...(row.id ? { id: row.id } : {}),
        billToCustomerId: row.billToCustomerId,
        method: row.method,
        percent: row.method === "Percent" && row.percent != null ? String(row.percent) : null,
        amount: row.method === "Amount" && row.amount != null ? String(row.amount) : null,
      })),
  };
}

/**
 * Changes who pays for one charge while the shipment is still in review. The
 * split is edited with the same editor as the shipment form; the server adds a
 * queue item for any payer who gains a share and cancels the item of any payer
 * left with nothing.
 */
export function BillingQueueReassignChargeDialog({
  open,
  onOpenChange,
  item,
  line,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: BillingQueueItem;
  line: PayerShareLine | null;
}) {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [, setSelection] = useQueryStates(queueSelectionSearchParamsParser);
  const shipment = item.shipment;

  const chargeTotal = line?.chargeTotal ?? null;
  const defaults = useMemo<ReassignFormValues>(
    () => ({ allocations: line ? allocationsForCharge(shipment, line) : [] }),
    [shipment, line],
  );
  const form = useForm<ReassignFormValues>({
    resolver: zodResolver(reassignFormSchema(chargeTotal)) as Resolver<ReassignFormValues>,
    defaultValues: defaults,
  });

  useEffect(() => {
    if (open) form.reset(defaults);
  }, [open, defaults, form]);

  const chargeName = line?.kind === "Freight" ? t("Freight") : (line?.description ?? "");
  const shipmentPayerId = shipment?.billToCustomerId || shipment?.customerId || "";
  const shipmentPayerName =
    item.payerShare?.payers.find((payer) => payer.id === shipmentPayerId)?.name ??
    shipment?.customer?.name ??
    t("customer");

  const mutation = useApiMutation<
    ReassignChargeResult,
    ReassignChargeInput,
    unknown,
    ReassignFormValues
  >({
    form,
    resourceName: "billing queue charge",
    mutationFn: (input) => apiService.billingQueueService.reassignCharge(item.id, input),
    onSuccess: (result) => {
      invalidate();
      toast.success(reassignSummary(t, chargeName, result));
      const next = nextItemAfterReassignment(item, result);
      if (next !== undefined) {
        void setSelection({ item: next });
      }
      onOpenChange(false);
    },
  });

  if (!line) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Change who pays for {0}", chargeName)}</DialogTitle>
          <DialogDescription>
            {t(
              "{0} of charges on shipment {1}. A payer who gains a share gets a queue item; a payer left with nothing has theirs canceled.",
              formatCurrency(chargeTotal ?? 0),
              shipment?.proNumber ?? item.shipmentId,
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form
            id="reassign-charge-form"
            onSubmit={form.handleSubmit((values) =>
              mutation.mutate(toReassignInput(line, values.allocations)),
            )}
          >
            <ChargeSplitEditor
              control={form.control as unknown as Control<FieldValues>}
              name="allocations"
              chargeAmount={chargeTotal}
              defaultPayer={
                shipmentPayerId ? { id: shipmentPayerId, label: shipmentPayerName } : null
              }
              disabled={mutation.isPending}
            />
            {form.formState.errors.root?.message ? (
              <p className="text-destructive text-2xs mt-2">{form.formState.errors.root.message}</p>
            ) : null}
          </form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form="reassign-charge-form" disabled={mutation.isPending}>
            {t("Save payers")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
