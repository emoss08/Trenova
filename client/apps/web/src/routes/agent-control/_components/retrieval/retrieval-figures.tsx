import { useNowSeconds } from "@/hooks/use-now-seconds";
import { formatUsd } from "@/lib/ai-usage-format";
import type { AIRetrievalStatus } from "@/lib/graphql/ai-retrieval";
import { formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Figs } from "../kit/layout";
import { budgetReached, monthCost, retrievalSentence, retrievalTotals } from "./retrieval-model";

/**
 * The index in five figures: what is indexed, what is waiting, what failed, what
 * retrieval has cost this month against the indexing budget, and when the indexer last
 * finished something.
 */
export function RetrievalFigures({ status }: { status: AIRetrievalStatus }) {
  const t = useT();
  const now = useNowSeconds();
  const totals = retrievalTotals(status.sources);
  const running = retrievalSentence(status).kind === "indexing";
  const unpriced = status.indexingUnpricedCalls + status.retrievalUnpricedCalls;
  const spent = Number(status.indexingCostMonthUsd);
  const budget = Number(status.settings.monthlyIndexingBudgetUsd);
  const share = budget > 0 && Number.isFinite(spent) ? Math.min(100, (spent / budget) * 100) : 0;

  return (
    <Figs
      items={[
        {
          label: t("Indexed"),
          value: formatNumber(totals.indexed),
          sub: t("searchable by meaning"),
        },
        {
          label: t("Waiting"),
          value: formatNumber(totals.waiting),
          tone: totals.waiting > 0 ? undefined : "dim",
          sub: running ? t("indexing now") : t("to be indexed"),
        },
        {
          label: t("Failed"),
          value: formatNumber(totals.failed),
          tone: totals.failed > 0 ? "t-d" : "dim",
          sub: totals.failed > 0 ? t("see below") : t("nothing failed"),
        },
        {
          label: t("Cost this month"),
          value: formatUsd(monthCost(status)) ?? "—",
          tone: budgetReached(status) ? "t-w" : undefined,
          sub: (
            <>
              <span className="ubar bud">
                <i style={{ width: `${share}%` }} />
              </span>
              {unpriced > 0
                ? t("{0, plural, one {# call had no price} other {# calls had no price}}", unpriced)
                : t("of {0} budget", formatUsd(status.settings.monthlyIndexingBudgetUsd) ?? "—")}
            </>
          ),
        },
        {
          label: t("Last run"),
          value: status.lastIndexedAt ? formatRelativeTime(status.lastIndexedAt - now) : "—",
          tone: status.lastIndexedAt ? undefined : "dim",
          sub: status.lastIndexedAt ? t("indexer round") : t("nothing indexed yet"),
        },
      ]}
    />
  );
}
