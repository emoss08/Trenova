import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueueService } from "@/services/billing-queue";
import type {
  BillingQueueApprovalRun,
  BulkRunAction,
  BulkRunParams,
} from "@trenova/shared/types/billing-queue";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { bulkRunCopy } from "./billing-queue-bulk-copy";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

const RUN_POLL_MS = 2000;

export function isRunFinished(run: BillingQueueApprovalRun): boolean {
  return run.status === "Completed" || run.status === "Failed" || run.status === "Undone";
}

export type StartBulkRun = (
  action: BulkRunAction,
  itemIds: string[],
  params?: BulkRunParams,
) => Promise<BillingQueueApprovalRun>;

/**
 * Runs one action over a selection as a server-side job. Every action but posting
 * waits out a short undo window before it writes, and the server checks each item
 * again when it does, so the toast offers Undo until the commit and then reports
 * what the run did, read from the run rather than from the list on screen.
 * `revertedRunIds` names the runs this hook saw undone or fail, so whatever was
 * set aside for one of them can come back.
 */
export function useBulkRun() {
  const t = useT();
  const invalidate = useInvalidateBillingQueue();
  const [runId, setRunId] = useState<string | null>(null);
  const [revertedRunIds, setRevertedRunIds] = useState<ReadonlySet<string>>(() => new Set());
  const announced = useRef(new Set<string>());
  const markReverted = (id: string) => setRevertedRunIds((current) => new Set([...current, id]));

  const undo = useApiMutation({
    mutationFn: (id: string) => billingQueueService.undoBulk(id),
    resourceName: "BillingQueueItem",
    onSuccess: (run) => {
      announced.current.add(run.id);
      setRunId(null);
      markReverted(run.id);
      toast.info(t("Undone; nothing was changed"));
    },
    onSettled: invalidate,
  });

  const start = useApiMutation({
    mutationFn: ({
      action,
      itemIds,
      params,
    }: {
      action: BulkRunAction;
      itemIds: string[];
      params: BulkRunParams;
    }) => billingQueueService.startBulk(action, itemIds, crypto.randomUUID(), params),
    resourceName: "BillingQueueItem",
    onSuccess: (run) => {
      setRunId(run.id);
      const copy = bulkRunCopy(run.action, t);
      const undoable = run.action !== "Post";
      toast(copy.running(run.totalCount), {
        id: `bulk-run-${run.id}`,
        description: t("Each item is checked again before it changes."),
        duration: undoable ? Math.max(1000, run.commitAt * 1000 - Date.now()) : 4000,
        action: undoable ? { label: t("Undo"), onClick: () => undo.mutate(run.id) } : undefined,
      });
    },
  });

  const { data: run } = useQuery({
    ...billingQueue.bulkRun(runId ?? ""),
    enabled: runId !== null,
    refetchInterval: (query) =>
      query.state.data && isRunFinished(query.state.data) ? false : RUN_POLL_MS,
  });

  useEffect(() => {
    if (!run || !isRunFinished(run) || announced.current.has(run.id)) {
      return;
    }
    announced.current.add(run.id);
    setRunId(null);
    invalidate();

    if (run.status === "Undone") {
      markReverted(run.id);
      return;
    }
    if (run.status === "Failed") {
      markReverted(run.id);
      toast.error(t("The bulk run failed"), { description: run.failureMessage ?? undefined });
      return;
    }
    const copy = bulkRunCopy(run.action, t);
    const missed = run.failedCount + run.skippedCount;
    if (missed === 0) {
      toast.success(copy.done(run.succeededCount));
      return;
    }
    toast.warning(copy.partial(run.succeededCount, run.totalCount), {
      description: t(
        "{0, plural, one {# item was left as it was — open it to see why} other {# items were left as they were — open them to see why}}",
        missed,
      ),
    });
  }, [run, invalidate, t]);

  const run_: StartBulkRun = (action, itemIds, params = {}) =>
    start.mutateAsync({ action, itemIds, params });

  return { start: run_, isStarting: start.isPending, revertedRunIds };
}
