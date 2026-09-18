import { Progress } from "@trenova/shared/components/ui/progress";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import type { BillingTransferRun } from "@/lib/graphql/billing-transfer";
import { isRunStopping, runProgress } from "./bulk-billing-transfer-run";

export function BulkBillingTransferResolving() {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 py-16 text-center" aria-live="polite">
      <Spinner className="size-5" />
      <p className="text-sm font-medium">{t("Finding eligible shipments...")}</p>
    </div>
  );
}

/** Opening straight onto a run, before it has been read back. */
export function BulkBillingTransferLoading() {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 py-16 text-center" aria-live="polite">
      <Spinner className="size-5" />
      <p className="text-sm font-medium">{t("Loading the transfer...")}</p>
    </div>
  );
}

export function BulkBillingTransferProgress({ run }: { run: BillingTransferRun }) {
  const t = useT();

  const progress = runProgress(run);
  const stopping = isRunStopping(run);

  // Queued, or still resolving a search: the total is genuinely not known yet,
  // so the bar is indeterminate rather than sitting empty at zero.
  if (progress === null) {
    return (
      <div className="flex flex-col items-center gap-3 py-16 text-center" aria-live="polite">
        <Spinner className="size-5" />
        <p className="text-sm font-medium">
          {run.status === "Queued"
            ? t("Starting the transfer...")
            : t("Finding eligible shipments...")}
        </p>
        <p className="text-muted-foreground text-xs">
          {t("You can close this window — the transfer keeps going.")}
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 py-10" aria-live="polite">
      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-4">
          <p className="text-sm font-medium">
            {stopping
              ? t("Finishing the batch in progress...")
              : t("Running the readiness check and transferring...")}
          </p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {t("{0} of {1} shipments checked", run.processedCount, run.totalCount)}
          </p>
        </div>
        <Progress
          value={run.processedCount}
          max={Math.max(run.totalCount, 1)}
          aria-label={t("Transfer progress")}
        />
      </div>
      <dl className="grid grid-cols-2 gap-3 text-xs">
        <div className="rounded-lg border px-3 py-2">
          <dt className="text-muted-foreground">{t("Transferred")}</dt>
          <dd className="text-lg font-semibold tabular-nums">{run.transferredCount}</dd>
        </div>
        <div className="rounded-lg border px-3 py-2">
          <dt className="text-muted-foreground">{t("Not transferred")}</dt>
          <dd className="text-lg font-semibold tabular-nums">{run.notTransferredCount}</dd>
        </div>
      </dl>
      <p className="text-muted-foreground text-xs">
        {t(
          "You can close this window. The transfer runs in the background and you will be notified when it finishes.",
        )}
      </p>
    </div>
  );
}
