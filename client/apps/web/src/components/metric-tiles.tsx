import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { ArrowDownIcon, ArrowUpIcon } from "lucide-react";
import { KpiStripItem } from "./kpi/kpi-strip";

/**
 * The vocabulary result panels share: a labelled figure, and a money delta
 * with its direction. Backtests and simulations both answer "what would this
 * change cost", and the answer has to read the same wherever it appears.
 */

export function formatDeltaPct(deltaPct: number): string {
  return `${deltaPct >= 0 ? "+" : ""}${deltaPct.toFixed(2)}%`;
}

export function DeltaValue({ delta, deltaPct }: { delta: number; deltaPct?: number }) {
  const isZero = delta === 0;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 font-mono tabular-nums",
        isZero
          ? "text-muted-foreground"
          : delta > 0
            ? "text-success-foreground"
            : "text-danger-foreground",
      )}
    >
      {!isZero &&
        (delta > 0 ? <ArrowUpIcon className="size-3" /> : <ArrowDownIcon className="size-3" />)}
      {formatCurrency(Math.abs(delta))}
      {deltaPct !== undefined && !isZero && (
        <span className="text-2xs opacity-80">({formatDeltaPct(deltaPct)})</span>
      )}
    </span>
  );
}

export function StatTile({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return <KpiStripItem label={label} value={<span className={tone}>{value}</span>} />;
}
