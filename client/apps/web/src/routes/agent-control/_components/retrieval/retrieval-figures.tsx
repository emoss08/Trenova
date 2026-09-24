import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { formatUsd } from "@/lib/ai-usage-format";
import type { AIRetrievalStatus } from "@/lib/graphql/ai-retrieval";
import { formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { budgetReached, monthCost, retrievalTotals } from "./retrieval-model";

const LAST_RUN_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

/**
 * The index in five figures: what is indexed, what is waiting, what failed,
 * what retrieval has cost this month, and when the indexer last finished
 * something.
 */
export function RetrievalFigures({ status }: { status: AIRetrievalStatus }) {
  const t = useT();
  const totals = retrievalTotals(status.sources);
  const unpriced = status.indexingUnpricedCalls + status.retrievalUnpricedCalls;
  const lastIndexedAt = status.lastIndexedAt;

  return (
    <KpiStrip aria-label={t("Retrieval figures")}>
      <KpiStripItem
        label={t("Indexed")}
        value={formatNumber(totals.indexed)}
        sub={t("searchable by meaning")}
      />
      <KpiStripItem
        label={t("Pending")}
        value={formatNumber(totals.waiting)}
        sub={t("waiting to be indexed")}
        tone={totals.waiting > 0 ? "info" : undefined}
      />
      <KpiStripItem
        label={t("Failed")}
        value={formatNumber(totals.failed)}
        sub={totals.failed > 0 ? t("listed below") : t("nothing failed")}
        tone={totals.failed > 0 ? "danger" : undefined}
      />
      <KpiStripItem
        label={t("Cost this month")}
        value={formatUsd(monthCost(status)) ?? "—"}
        sub={
          unpriced > 0
            ? t("{0, plural, one {# call had no price} other {# calls had no price}}", unpriced)
            : t(
                "{0} of {1} indexing budget",
                formatUsd(status.indexingCostMonthUsd) ?? "—",
                formatUsd(status.settings.monthlyIndexingBudgetUsd) ?? "—",
              )
        }
        tone={budgetReached(status) ? "warning" : undefined}
      />
      <KpiStripItem
        label={t("Last run")}
        value={
          lastIndexedAt ? formatRelativeTime(lastIndexedAt - Math.floor(Date.now() / 1000)) : "—"
        }
        sub={
          lastIndexedAt
            ? formatUnixInUserTimezone(lastIndexedAt, LAST_RUN_FORMAT)
            : t("Nothing indexed yet")
        }
      />
    </KpiStrip>
  );
}
