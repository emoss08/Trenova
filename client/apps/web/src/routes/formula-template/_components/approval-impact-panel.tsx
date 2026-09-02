import { queries } from "@/lib/queries";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { BacktestResult } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { MinusIcon, TrendingDownIcon, TrendingUpIcon } from "lucide-react";

const TOP_MOVERS_SHOWN = 5;

function formatSignedCurrency(amount: number): string {
  const formatted = formatCurrency(Math.abs(amount));
  return amount < 0 ? `-${formatted}` : `+${formatted}`;
}

function MoverRow({ result }: { result: BacktestResult }) {
  const increased = result.delta > 0;
  return (
    <div className="flex items-center justify-between gap-2 px-3 py-1.5 text-xs">
      <span className="truncate font-mono">{result.proNumber || result.shipmentId}</span>
      <div className="flex shrink-0 items-center gap-2 tabular-nums">
        <span className="text-muted-foreground">
          {formatCurrency(result.currentAmount)} → {formatCurrency(result.candidateAmount)}
        </span>
        <span
          className={cn(
            "font-medium",
            increased ? "text-emerald-600 dark:text-emerald-400" : "text-destructive",
          )}
        >
          {formatSignedCurrency(result.delta)}
        </span>
      </div>
    </div>
  );
}

export function ApprovalImpactPanel({ templateId }: { templateId: string }) {
  const { data, isLoading, isError } = useQuery({
    ...queries.formulaTemplate.approvalImpact(templateId),
    enabled: !!templateId,
    staleTime: 30_000,
  });

  if (isLoading) {
    return (
      <div className="space-y-1.5 rounded-md border p-3">
        <Skeleton className="h-3.5 w-40" />
        <Skeleton className="h-3 w-52" />
        <Skeleton className="h-3 w-48" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        Impact analysis is unavailable right now. You can still approve.
      </div>
    );
  }

  const { summary, results } = data;

  if (summary.shipmentCount === 0) {
    return (
      <div className="text-muted-foreground rounded-md border px-3 py-2 text-xs">
        No shipments have been rated with this template yet, so approving has no effect on existing
        pricing.
      </div>
    );
  }

  const noChange = summary.changedCount === 0 && summary.errorCount === 0;
  const totalIncreased = summary.totalDelta > 0;
  const TrendIcon = noChange ? MinusIcon : totalIncreased ? TrendingUpIcon : TrendingDownIcon;

  // Results arrive biggest-movers-first from the server; only re-rated rows
  // that actually moved are worth listing.
  const movers = results
    .filter((result) => !result.currentError && !result.candidateError && result.delta !== 0)
    .slice(0, TOP_MOVERS_SHOWN);

  return (
    <div className="overflow-hidden rounded-md border">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <span className="text-xs font-semibold">Impact on recent shipments</span>
        <span className="text-2xs text-muted-foreground">
          last {summary.shipmentCount} rated with this template
        </span>
      </div>

      <div className="flex items-center gap-2 px-3 py-2">
        <TrendIcon
          className={cn(
            "size-4 shrink-0",
            noChange
              ? "text-muted-foreground"
              : totalIncreased
                ? "text-emerald-600 dark:text-emerald-400"
                : "text-destructive",
          )}
        />
        {noChange ? (
          <span className="text-xs">
            Re-rating produces identical charges — this change is pricing-neutral for existing
            traffic.
          </span>
        ) : (
          <span className="text-xs">
            Re-rating would move totals by{" "}
            <span
              className={cn(
                "font-medium tabular-nums",
                totalIncreased ? "text-emerald-600 dark:text-emerald-400" : "text-destructive",
              )}
            >
              {formatSignedCurrency(summary.totalDelta)}
            </span>{" "}
            ({summary.changedCount} of {summary.evaluatedCount} shipments change
            {summary.errorCount > 0 && (
              <span className="text-destructive">, {summary.errorCount} fail to evaluate</span>
            )}
            ).
          </span>
        )}
      </div>

      {movers.length > 0 && (
        <div className="divide-y border-t">
          {movers.map((result) => (
            <MoverRow key={result.shipmentId} result={result} />
          ))}
        </div>
      )}
    </div>
  );
}
