import type { BillingTransferRun } from "@/lib/graphql/billing-transfer";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatDuration, formatNumber } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  isRunStopping,
  isRunTerminal,
  runElapsedSeconds,
  runSecondsLeft,
} from "./bulk-billing-transfer-run";
import { BulkBillingTransferStage } from "./bulk-billing-transfer-stage";

/**
 * A run while it works: what it is doing and how long is left, over the stage that draws
 * each answer as it arrives. `onSettled` is passed through from the stage, which calls it once
 * a finished run's last bill has landed.
 */
export function BulkBillingTransferProgress({
  run,
  onSettled,
}: {
  run: BillingTransferRun;
  onSettled: () => void;
}) {
  const t = useT();
  const now = useNow("second");

  const stopping = isRunStopping(run);
  const secondsLeft = stopping ? null : runSecondsLeft(run, now);
  const elapsed = runElapsedSeconds(run, now);

  return (
    <div className="flex flex-col gap-3.5">
      <div className="flex items-baseline justify-between gap-4 text-xs">
        <span aria-live="polite" className="min-w-0">
          <TextShimmer as="span" className="font-medium" duration={2.4}>
            {activityLabel(run, stopping, t)}
          </TextShimmer>
        </span>
        <span className="text-muted-foreground shrink-0 tabular-nums">
          {isRunTerminal(run)
            ? t("Finishing up")
            : secondsLeft !== null
              ? t("About {0} left", formatDuration(secondsLeft))
              : elapsed !== null
                ? t("Running for {0}", formatDuration(elapsed))
                : t("Waiting for a worker")}
        </span>
        {run.totalCount > 0 ? (
          <span className="sr-only">
            {t(
              "{0} of {1} shipments checked",
              formatNumber(run.processedCount),
              formatNumber(run.totalCount),
            )}
          </span>
        ) : null}
      </div>
      <BulkBillingTransferStage key={run.id} run={run} onSettled={onSettled} />
    </div>
  );
}

function activityLabel(run: BillingTransferRun, stopping: boolean, t: TranslateFn): string {
  if (stopping) return t("Finishing the batch in progress");
  if (run.status === "Queued") return t("Starting the transfer");
  if (run.totalCount <= 0) return t("Finding eligible shipments");
  return run.processedCount === 0
    ? t("Running the readiness check")
    : t("Checking readiness and transferring");
}
