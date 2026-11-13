import { useT } from "@trenova/shared/i18n/use-t";
import { WithheldPanel } from "@trenova/shared/components/withheld-value";
import { cn, formatCompactCurrency, formatCurrency } from "@trenova/shared/lib/utils";
import type { ShipmentAnalyticsData } from "@/lib/shipment-analytics";

type CustomerMixCard = NonNullable<ShipmentAnalyticsData["customerMix"]>;
type CustomerMixEntry = CustomerMixCard["entries"][number];

type CustomerMixProps = {
  /** Null when the reader cannot read customers. */
  customerMix: CustomerMixCard | null;
};

const SHARE_BAR_COLORS = [
  "var(--chart-1)",
  "var(--chart-3)",
  "var(--chart-2)",
  "var(--chart-4)",
  "var(--chart-8)",
];

/**
 * Who the revenue comes from, largest first, each with its share and how it is
 * trending. It is a widget's body, not a card: the widget that shows it draws the
 * frame and the title, and names the window in its own header.
 */
export function CustomerMix({ customerMix }: CustomerMixProps) {
  const t = useT();

  if (!customerMix) {
    return (
      <WithheldPanel>{t("You need access to customers to see the customer mix")}</WithheldPanel>
    );
  }

  return <CustomersList entries={customerMix.entries} />;
}

function CustomersList({ entries }: { entries: CustomerMixEntry[] }) {
  const t = useT();

  if (entries.length === 0) {
    return <EmptyState label={t("No customer revenue in this window")} />;
  }

  return (
    <ul className="flex flex-col gap-2.5">
      {entries.map((entry, index) => (
        <li key={entry.customerId} className="flex items-center gap-2">
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <span className="truncate text-xs font-medium">{entry.name}</span>
              <span
                className="text-muted-foreground font-mono text-2xs tabular-nums"
                title={t(
                  "{0} · {1, plural, one {# load} other {# loads}}",
                  formatCurrency(entry.revenue),
                  entry.loads,
                )}
              >
                {formatCompactCurrency(entry.revenue)} · {entry.loads}
              </span>
            </div>
            <div aria-hidden className="mt-1 h-1 w-full overflow-hidden rounded-full bg-muted">
              <span
                className="block h-full rounded-full"
                style={{
                  width: `${Math.min(100, entry.share)}%`,
                  background: SHARE_BAR_COLORS[index] ?? SHARE_BAR_COLORS.at(-1),
                }}
              />
            </div>
          </div>
          <CustomerTrend trend={entry.trend} />
        </li>
      ))}
    </ul>
  );
}

function CustomerTrend({ trend }: { trend: number }) {
  return (
    <span
      className={cn(
        "w-10 text-right font-mono text-2xs tabular-nums",
        trend > 0 && "text-success",
        trend < 0 && "text-destructive",
        trend === 0 && "text-muted-foreground",
      )}
    >
      {trend > 0 ? "▲" : trend < 0 ? "▼" : "–"}
      {formatPercent(Math.abs(trend))}
    </span>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="text-muted-foreground flex h-full min-h-32 items-center justify-center px-3 py-6 text-center text-xs">
      {label}
    </div>
  );
}

function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
  }).format(value);
}
