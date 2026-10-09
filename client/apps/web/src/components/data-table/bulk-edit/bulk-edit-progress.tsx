import { useApiMutation } from "@/hooks/use-api-mutation";
import { BULK_EDIT_DONE_STATUSES, undoMyBulkEdit, type BulkEditJob } from "@/lib/graphql/bulk-edit";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FlipBackwardIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";

/** Asked again this often while a job runs, in case its pushed progress is missed. */
const FALLBACK_POLL_MS = 3000;

type BulkEditProgressProps = {
  jobId: string;
  onDismiss: () => void;
  /** Runs once when the job finishes, so the table can refetch its rows. */
  onDone: (job: BulkEditJob) => void;
};

/**
 * How far a bulk edit has got. Progress arrives as the server pushes it; when the
 * edit finishes the card says what changed, lists the first rows that could not be,
 * and offers to put everything back.
 */
export function BulkEditProgress({ jobId, onDismiss, onDone }: BulkEditProgressProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [showFailures, setShowFailures] = useState(false);
  const jobQuery = useQuery({
    ...queries.bulkEdit.job(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      return status && BULK_EDIT_DONE_STATUSES.includes(status) ? false : FALLBACK_POLL_MS;
    },
  });
  const job = jobQuery.data;
  const done = !!job && BULK_EDIT_DONE_STATUSES.includes(job.status);

  const reportedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!job || !done) return;
    const key = `${job.id}:${job.status}`;
    if (reportedRef.current === key) return;
    reportedRef.current = key;
    onDone(job);
  }, [job, done, onDone]);

  const { mutate: undo, isPending: undoing } = useApiMutation<BulkEditJob, undefined>({
    mutationFn: () => undoMyBulkEdit(jobId),
    resourceName: "Bulk edit",
    onSuccess: (updated) => {
      queryClient.setQueryData(queries.bulkEdit.job(jobId).queryKey, updated);
    },
  });

  if (!job) return null;

  const percent = job.totalCount > 0 ? Math.round((job.processedCount / job.totalCount) * 100) : 0;
  const undoingNow = job.status === "Undoing";
  const title = undoingNow
    ? t("Putting rows back")
    : job.status === "Undone"
      ? t("Rows put back")
      : job.status === "Failed"
        ? t("Bulk edit stopped")
        : done
          ? t("Bulk edit finished")
          : t("Editing rows");

  return (
    <div
      role="status"
      aria-live="polite"
      className="ui-lift-float border-border bg-background fixed right-6 bottom-6 z-50 w-80 rounded-lg border p-3"
    >
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">{title}</p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {t(
              "{0} of {1} · {2} changed · {3} failed",
              job.processedCount.toLocaleString(),
              job.totalCount.toLocaleString(),
              job.changedCount.toLocaleString(),
              job.failedCount.toLocaleString(),
            )}
          </p>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("Close")}
          onClick={onDismiss}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      </div>
      <div
        className="bg-muted mt-2 h-1.5 overflow-hidden rounded-full"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
      >
        <div
          className={cn(
            "h-full rounded-full transition-[width] duration-300",
            job.status === "Failed" ? "bg-danger" : "bg-brand",
          )}
          style={{ width: `${done ? 100 : percent}%` }}
        />
      </div>
      {job.failureMessage ? (
        <p className="text-danger-foreground mt-2 text-xs">{job.failureMessage}</p>
      ) : null}
      {job.failures.length > 0 ? (
        <div className="mt-2">
          <Button
            type="button"
            variant="link"
            size="xs"
            className="h-auto p-0 text-xs"
            onClick={() => setShowFailures((open) => !open)}
          >
            {showFailures ? t("Hide rows that failed") : t("Show rows that failed")}
          </Button>
          {showFailures ? (
            <ScrollArea viewportClassName="max-h-32" className="mt-1">
              <ul className="flex flex-col gap-1 text-xs">
                {job.failures.map((failure) => (
                  <li key={failure.id} className="flex gap-2">
                    <span className="text-muted-foreground shrink-0 font-mono">{failure.id}</span>
                    <span className="min-w-0 truncate" title={failure.message}>
                      {failure.message}
                    </span>
                  </li>
                ))}
              </ul>
            </ScrollArea>
          ) : null}
        </div>
      ) : null}
      {job.canUndo ? (
        <div className="mt-2 flex justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            isLoading={undoing}
            loadingText={t("Undoing...")}
            onClick={() => undo(undefined)}
          >
            <FlipBackwardIcon className="size-3.5" />
            {t("Undo")}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
