import { nextItemAfterReassignment, reassignSummary } from "@/lib/billing-queue-charges";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import type { BillingQueueItem, ReassignChargeResult } from "@trenova/shared/types/billing-queue";
import { useQueryStates } from "nuqs";
import { queueSelectionSearchParamsParser } from "../use-billing-queue-state";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

/**
 * What follows a charge moving between payers, wherever it was moved from: the queue
 * reloads, the biller is told which items it added or canceled, and an item the move
 * canceled gives way to the next one.
 */
export function useAfterReassignment(item: BillingQueueItem) {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [, setSelection] = useQueryStates(queueSelectionSearchParamsParser);

  return (result: ReassignChargeResult, chargeName: string) => {
    invalidate();
    toast.success(reassignSummary(t, chargeName, result));
    const next = nextItemAfterReassignment(item, result);
    if (next !== undefined) {
      void setSelection({ item: next });
    }
  };
}
