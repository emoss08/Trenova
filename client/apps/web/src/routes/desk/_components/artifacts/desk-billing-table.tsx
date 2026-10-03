import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useDeskStore } from "@/stores/desk-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { BillingQueueSummary } from "@trenova/shared/types/billing-queue";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import {
  bulkSplit,
  itemStage,
  selectionState,
  stageLabel,
  undoSecondsLeft,
} from "./billing-item-checks";
import { BiIcon } from "./desk-billing-item";
import { billingCardLineage, useDeskBillingStore } from "./desk-billing-store";

/** At most this many rows' live state is read at once, the server's own limit. */
const SUMMARY_LIMIT = 200;

/**
 * The live state of a billing queue table's rows. The table is what the agent
 * read when it answered; the rows' status and what they still need are read
 * now, and stay current as the queue changes.
 */
export function useBillingSummaries(ids: readonly string[], enabled: boolean) {
  const wanted = useMemo(() => ids.slice(0, SUMMARY_LIMIT), [ids]);
  const { data } = useQuery({
    ...queries.billingQueue.summaries([...wanted]),
    enabled: enabled && wanted.length > 0,
  });

  return useMemo(
    () => new Map<string, BillingQueueSummary>((data ?? []).map((row) => [row.id, row])),
    [data],
  );
}

/** A row's status as the queue reads it now, with how many checks need a person. */
export function BillingRowPill({ summary }: { summary: BillingQueueSummary }) {
  const t = useT();
  const stage = itemStage(summary.status);
  const dot =
    stage === "review"
      ? "dk-blue"
      : stage === "approved"
        ? "dk-green"
        : stage === "posted"
          ? "dk-ink"
          : stage === "held"
            ? "dk-warn"
            : "";
  return (
    <span className={cn("dk-ax-pill", dot)}>
      <i />
      {stage === "held" ? t("On hold") : stageLabel(summary.status, summary.holdReasonCode, t)}
      {stage === "review" && summary.needsCount > 0 && (
        <em className="dk-ax-nd">{summary.needsCount}</em>
      )}
    </span>
  );
}

export function BillingCheckbox({
  state,
  label,
  onClick,
}: {
  state: "all" | "some" | "none" | boolean;
  label: string;
  onClick?: () => void;
}) {
  const on = state === true || state === "all";
  const mid = state === "some";
  const box = (
    <span className={cn("dk-ax-cb", on && "dk-on", mid && "dk-mid")}>
      <BiIcon name="check" size={10} stroke={3} />
    </span>
  );
  if (!onClick) return box;

  return (
    <button
      type="button"
      className={cn("dk-ax-cb", on && "dk-on", mid && "dk-mid")}
      aria-label={label}
      aria-pressed={mid ? "mixed" : on}
      title={label}
      onClick={onClick}
    >
      <BiIcon name="check" size={10} stroke={3} />
    </button>
  );
}

/**
 * Opens a queue row's item: in the conversation's billing item artifact when
 * it has one, which then shows this row's item, or else in place of the table.
 */
export function useOpenBillingItem(threadId: string, onInline: (itemId: string) => void) {
  const queryClient = useQueryClient();
  const select = useDeskBillingStore((state) => state.select);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);

  return useCallback(
    (itemId: string) => {
      const lineageId = billingCardLineage(queryClient, threadId);
      if (lineageId) {
        select(lineageId, itemId);
        setActiveArtifact(threadId, lineageId);
        return;
      }
      onInline(itemId);
    },
    [onInline, queryClient, select, setActiveArtifact, threadId],
  );
}

type Undo = { runId: string; commitAt: number; count: number };

/**
 * The floating bar over a queue table with rows picked: what the picked rows
 * come to, Review for the first that needs a person, and Approve for the
 * ready ones. Approving starts one job on the server that waits out a short
 * undo window before it writes; the bar counts the window down, and Undo
 * stops the job before it touches anything.
 */
export function BillingBulkBar({
  selected,
  summaries,
  onClear,
  onReview,
}: {
  selected: string[];
  summaries: ReadonlyMap<string, BillingQueueSummary>;
  onClear: () => void;
  onReview: (itemId: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [undo, setUndo] = useState<Undo | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const split = bulkSplit(selected, summaries);

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["billingQueue"] });
    void queryClient.invalidateQueries({ queryKey: ["billing-queue-list"] });
  }, [queryClient]);

  // The undo shows only until the server commits; past that the run's own
  // results take over.
  const left = undo ? undoSecondsLeft(undo.commitAt, now) : 0;
  const counting = undo !== null && left > 0;
  useEffect(() => {
    if (!undo) return;
    const timer = window.setInterval(() => {
      const at = Date.now();
      setNow(at);
      if (undoSecondsLeft(undo.commitAt, at) <= 0) {
        window.clearInterval(timer);
        refresh();
      }
    }, 1000);
    return () => window.clearInterval(timer);
  }, [undo, refresh]);

  const start = useApiMutation({
    mutationFn: (itemIds: string[]) =>
      apiService.billingQueueService.startBulkApprove(itemIds, crypto.randomUUID()),
    resourceName: "BillingQueueItem",
    onSuccess: (run) => {
      setNow(Date.now());
      setUndo({ runId: run.id, commitAt: run.commitAt, count: run.totalCount });
      onClear();
    },
  });
  const cancel = useApiMutation({
    mutationFn: (runId: string) => apiService.billingQueueService.undoBulkApprove(runId),
    resourceName: "BillingQueueItem",
    onSettled: () => {
      setUndo(null);
      refresh();
    },
  });

  // After the window, the run's results arrive over the event stream; items
  // that could not be approved are said once.
  const { data: run } = useQuery({
    ...queries.billingQueue.approvalRun(undo?.runId ?? ""),
    enabled: undo !== null && !counting,
  });
  const announced = useRef(new Set<string>());
  useEffect(() => {
    if (!run || announced.current.has(run.id) || !["Completed", "Failed"].includes(run.status)) {
      return;
    }
    announced.current.add(run.id);
    const missed = run.failedCount + run.skippedCount;
    if (missed > 0) {
      toast.warning(
        t(
          "{0, plural, one {# invoice wasn't approved} other {# invoices weren't approved}} — open it to see why",
          missed,
        ),
      );
    }
  }, [run, t]);

  if (!counting && selected.length === 0) {
    return null;
  }

  return (
    <div className="dk-ax-bulk" key={counting ? "undo" : "select"}>
      {counting && undo ? (
        <>
          <span className="dk-ax-bk-ok">
            <BiIcon name="check" size={12} stroke={2.6} />
          </span>
          <span className="dk-ax-bk-t">
            <b>
              {t("{0, plural, one {Approved # invoice} other {Approved # invoices}}", undo.count)}
            </b>
          </span>
          <button
            type="button"
            className="dk-ax-btn"
            disabled={cancel.isPending}
            onClick={() => cancel.mutate(undo.runId)}
          >
            {t("Undo")} <em className="dk-ax-bk-n">{left}</em>
          </button>
        </>
      ) : (
        <>
          <span className="dk-ax-bk-t">
            <b>{t("{0} selected", selected.length)}</b>
            <span>
              {[
                t("{0} ready", split.ready.length),
                split.needs.length > 0 ? t("{0} need you", split.needs.length) : "",
                split.other > 0 ? t("{0} already done or held", split.other) : "",
              ]
                .filter(Boolean)
                .join(" · ")}
            </span>
          </span>
          <button type="button" className="dk-ax-btn dk-ghost" onClick={onClear}>
            {t("Clear")}
          </button>
          {split.needs.length > 0 && (
            <button type="button" className="dk-ax-btn" onClick={() => onReview(split.needs[0])}>
              {t("Review {0}", split.needs.length)}
            </button>
          )}
          <button
            type="button"
            className="dk-ax-btn dk-ink"
            disabled={split.ready.length === 0 || start.isPending}
            onClick={() => start.mutate(split.ready)}
          >
            {split.ready.length > 0 ? t("Approve {0}", split.ready.length) : t("Approve")}
          </button>
        </>
      )}
    </div>
  );
}

export { selectionState };
