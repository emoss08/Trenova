import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { BillingQueueApprovalRun } from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

const RUN_POLL_MS = 2000;

function isFinished(run: BillingQueueApprovalRun): boolean {
  return run.status === "Completed" || run.status === "Failed" || run.status === "Undone";
}

/**
 * Approves a selection as one server-side run. The server waits out a short
 * undo window before it writes and checks every item again when it does, so
 * the toast offers Undo until the commit and then reports what was approved
 * and what was not, read from the run rather than from the list on screen.
 */
export function useBulkApprove() {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [runId, setRunId] = useState<string | null>(null);
  const announced = useRef(new Set<string>());

  const undo = useApiMutation({
    mutationFn: (id: string) => apiService.billingQueueService.undoBulkApprove(id),
    resourceName: "BillingQueueItem",
    onSuccess: (run) => {
      announced.current.add(run.id);
      setRunId(null);
      toast.info(t("Approval undone; nothing was changed"));
    },
    onSettled: invalidate,
  });

  const start = useApiMutation({
    mutationFn: ({ itemIds, assignMe }: { itemIds: string[]; assignMe: boolean }) =>
      apiService.billingQueueService.startBulkApprove(itemIds, crypto.randomUUID(), assignMe),
    resourceName: "BillingQueueItem",
    onSuccess: (run) => {
      setRunId(run.id);
      toast(t("{0, plural, one {Approving # item} other {Approving # items}}", run.totalCount), {
        id: `bulk-approve-${run.id}`,
        description: t("Each item is checked again before it is approved."),
        duration: Math.max(1000, run.commitAt * 1000 - Date.now()),
        action: { label: t("Undo"), onClick: () => undo.mutate(run.id) },
      });
    },
  });

  const { data: run } = useQuery({
    queryKey: ["billingQueue", "approvalRun", runId],
    queryFn: () => apiService.billingQueueService.getBulkApprove(runId ?? ""),
    enabled: runId !== null,
    refetchInterval: (query) =>
      query.state.data && isFinished(query.state.data) ? false : RUN_POLL_MS,
  });

  useEffect(() => {
    if (!run || !isFinished(run) || announced.current.has(run.id)) {
      return;
    }
    announced.current.add(run.id);
    setRunId(null);
    invalidate();

    if (run.status === "Undone") {
      return;
    }
    if (run.status === "Failed") {
      toast.error(t("The approval run failed"), {
        description: run.failureMessage ?? undefined,
      });
      return;
    }
    const missed = run.failedCount + run.skippedCount;
    if (missed === 0) {
      toast.success(
        t("{0, plural, one {Approved # item} other {Approved # items}}", run.approvedCount),
      );
      return;
    }
    toast.warning(t("Approved {0} of {1}", run.approvedCount, run.totalCount), {
      description: t(
        "{0, plural, one {# item wasn't approved — open it to see why} other {# items weren't approved — open them to see why}}",
        missed,
      ),
    });
  }, [run, invalidate, t]);

  return {
    approve: (itemIds: string[], assignMe: boolean) => start.mutateAsync({ itemIds, assignMe }),
    isStarting: start.isPending,
  };
}
