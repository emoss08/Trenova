import { Progress } from "@trenova/shared/components/ui/progress";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import type { BulkBillingTransferResult } from "@/lib/graphql/billing-transfer";

export function BulkBillingTransferResolving() {
  const t = useT();

  return (
    <div className="flex flex-col items-center gap-3 py-16 text-center" aria-live="polite">
      <Spinner className="size-5" />
      <p className="text-sm font-medium">{t("Finding eligible shipments...")}</p>
    </div>
  );
}

export function BulkBillingTransferProgress({
  totalCount,
  processedCount,
  results,
}: {
  totalCount: number;
  processedCount: number;
  results: BulkBillingTransferResult[];
}) {
  const t = useT();

  const transferred = results.reduce((count, result) => (result.success ? count + 1 : count), 0);
  const notTransferred = results.length - transferred;

  return (
    <div className="flex flex-col gap-4 py-10" aria-live="polite">
      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-4">
          <p className="text-sm font-medium">
            {t("Running the readiness check and transferring...")}
          </p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {t("{0} of {1} shipments checked", processedCount, totalCount)}
          </p>
        </div>
        <Progress
          value={processedCount}
          max={Math.max(totalCount, 1)}
          aria-label={t("Transfer progress")}
        />
      </div>
      <dl className="grid grid-cols-2 gap-3 text-xs">
        <div className="rounded-lg border px-3 py-2">
          <dt className="text-muted-foreground">{t("Transferred")}</dt>
          <dd className="text-lg font-semibold tabular-nums">{transferred}</dd>
        </div>
        <div className="rounded-lg border px-3 py-2">
          <dt className="text-muted-foreground">{t("Not transferred")}</dt>
          <dd className="text-lg font-semibold tabular-nums">{notTransferred}</dd>
        </div>
      </dl>
      <p className="text-muted-foreground text-xs">
        {t(
          "Keep this window open. Shipments are sent in batches, and stopping takes effect once the current batch finishes.",
        )}
      </p>
    </div>
  );
}
