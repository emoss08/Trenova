import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { BacktestResponse } from "@trenova/shared/types/formula-template";
import { formatMoney } from "../model/format";

const MOVERS_SHOWN = 3;

function signedMoney(amount: number): string {
  return `${amount >= 0 ? "+" : "−"}${formatMoney(Math.abs(amount), 0)}`;
}

/**
 * What the saved content does to recent freight against the version that
 * rates it now: how many shipments were re-rated, how many would change, the
 * revenue it moves, and the three that move most. Every figure is the
 * backtest's.
 */
export function ImpactSummary({
  data,
  isLoading,
  isError,
  showFailures = false,
}: {
  data: BacktestResponse | undefined;
  isLoading: boolean;
  isError: boolean;
  showFailures?: boolean;
}) {
  const t = useT();

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-7 w-full" />
        <Skeleton className="h-7 w-full" />
      </div>
    );
  }
  if (isError || !data) {
    return (
      <p className="text-foreground-muted rounded-md border px-3 py-2 text-xs">
        {t("The impact could not be worked out right now. Review still covers what is saved.")}
      </p>
    );
  }

  const { summary, results } = data;
  if (summary.shipmentCount === 0) {
    return (
      <p className="text-foreground-muted rounded-md border px-3 py-2 text-xs">
        {t("Nothing has rated with this template yet, so no recent shipment would change.")}
      </p>
    );
  }

  const movers = results
    .filter((result) => !result.currentError && !result.candidateError && result.delta !== 0)
    .slice(0, MOVERS_SHOWN);

  return (
    <div className="flex flex-col gap-2">
      <KpiStrip minItemWidth="6.5rem">
        <KpiStripItem label={t("Re-rated")} value={summary.evaluatedCount} />
        <KpiStripItem label={t("Would change")} value={summary.changedCount} />
        {showFailures && (
          <KpiStripItem
            label={t("Fail to rate")}
            value={summary.candidateErrorCount}
            tone={summary.candidateErrorCount > 0 ? "danger" : undefined}
          />
        )}
        <KpiStripItem
          label={t("Revenue")}
          value={signedMoney(summary.totalDelta)}
          sub={t(
            "{0}%",
            `${summary.totalDeltaPct >= 0 ? "+" : ""}${summary.totalDeltaPct.toFixed(1)}`,
          )}
          tone={summary.totalDelta > 0 ? "success" : summary.totalDelta < 0 ? "danger" : undefined}
        />
      </KpiStrip>
      {movers.length > 0 ? (
        <div className="flex flex-col overflow-hidden rounded-md border">
          {movers.map((mover) => (
            <div
              key={mover.shipmentId}
              className="flex items-center justify-between gap-2 border-b px-3 py-1.5 text-xs last:border-b-0"
            >
              <span className="truncate font-mono">{mover.proNumber || mover.shipmentId}</span>
              <span className="font-mono tabular-nums">
                {formatMoney(mover.currentAmount)} →{" "}
                <b className="font-semibold">{formatMoney(mover.candidateAmount)}</b>
              </span>
            </div>
          ))}
        </div>
      ) : (
        <p className="text-foreground-muted text-xs">
          {t("No recent shipment would rate differently.")}
        </p>
      )}
    </div>
  );
}
