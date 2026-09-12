import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { OpenStatement } from "@trenova/shared/types/statement";

const WEEK_SECONDS = 7 * 86_400;

type StatementMetrics = {
  open: number;
  shipments: number;
  value: number;
  currency: string;
  billingThisWeek: number;
  held: number;
};

/**
 * Rolls the list up once so the strip and the sidebar cannot disagree.
 *
 * Currency is taken from the first statement that has freight on it rather than
 * summed blindly across currencies: a deployment billing in two currencies would
 * otherwise show a meaningless total. Mixed currencies are rare enough that the
 * honest fix is to show the dominant one, not to invent an exchange rate.
 */
export function summarizeStatements(
  statements: readonly OpenStatement[],
  nowSeconds: number,
): StatementMetrics {
  const metrics: StatementMetrics = {
    open: 0,
    shipments: 0,
    value: 0,
    currency: "USD",
    billingThisWeek: 0,
    held: 0,
  };

  for (const statement of statements) {
    if (statement.shipmentCount === 0) continue;

    metrics.open += 1;
    metrics.shipments += statement.shipmentCount;
    metrics.value += Number(statement.totalAmount ?? 0);
    if (metrics.open === 1) metrics.currency = statement.currencyCode;
    if (statement.periodEnd - nowSeconds <= WEEK_SECONDS) metrics.billingThisWeek += 1;
    if (statement.belowMinimum) metrics.held += 1;
  }

  return metrics;
}

export function StatementKPIStrip({
  statements,
  nowSeconds,
}: {
  statements: readonly OpenStatement[];
  nowSeconds: number;
}) {
  const metrics = summarizeStatements(statements, nowSeconds);

  const tiles: { key: string; label: string; value: string; muted?: boolean }[] = [
    { key: "open", label: "Accruing", value: String(metrics.open) },
    { key: "shipments", label: "Shipments", value: String(metrics.shipments) },
    {
      key: "value",
      label: "Value",
      value: formatCurrency(metrics.value, metrics.currency),
    },
    { key: "week", label: "Bills this week", value: String(metrics.billingThisWeek) },
  ];

  if (metrics.held > 0) {
    tiles.push({
      key: "held",
      label: "Under minimum",
      value: String(metrics.held),
      muted: true,
    });
  }

  return (
    <div className="mx-4 mt-3">
      <div className="bg-card flex items-center rounded-lg border">
        {tiles.map((tile, index) => (
          <div
            key={tile.key}
            className={cn("flex items-center gap-2 px-4 py-2.5", index > 0 && "border-l")}
          >
            <span className="text-muted-foreground text-xs">{tile.label}</span>
            <span
              className={cn(
                "text-sm font-semibold tabular-nums",
                tile.muted && "text-muted-foreground",
              )}
            >
              {tile.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
