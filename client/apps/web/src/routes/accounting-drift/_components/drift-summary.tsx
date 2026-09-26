import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import {
  tableFilterSearchParamsParser,
  tablePaginationSearchParamsParser,
} from "@/hooks/data-table/use-data-table-state";
import type { AccountingDriftOverview } from "@/lib/graphql/accounting-drift";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { activeDriftBucket, driftBucketFilter, type DriftBucket } from "./drift-filters";

export function DriftSummary({ overview }: { overview: AccountingDriftOverview }) {
  const t = useT();
  const [filters, setFilters] = useQueryStates({
    ...tableFilterSearchParamsParser,
    ...tablePaginationSearchParamsParser,
  });
  const current = activeDriftBucket(filters.fieldFilters);
  const { summary } = overview;

  const select = (bucket: DriftBucket) => {
    void setFilters({
      fieldFilters: current === bucket ? [] : driftBucketFilter(bucket),
      pageIndex: 1,
    });
  };

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("Open differences")}
        value={summary.open}
        tone={summary.open > 0 ? "warning" : undefined}
        active={current === "open"}
        onClick={() => select("open")}
      />
      <KpiStripItem
        label={t("Different amounts")}
        value={summary.amountOpen}
        sub={t("totals and customer balances")}
        active={current === "amount"}
        onClick={() => select("amount")}
      />
      <KpiStripItem
        label={t("Deleted or voided")}
        value={summary.goneOpen}
        sub={t("in the books")}
        tone={summary.goneOpen > 0 ? "danger" : undefined}
        active={current === "gone"}
        onClick={() => select("gone")}
      />
      <KpiStripItem
        label={t("Settled")}
        value={summary.resolvedSince}
        sub={t("last 7 days")}
        active={current === "settled"}
        onClick={() => select("settled")}
      />
    </KpiStrip>
  );
}

export function DriftSummarySkeleton() {
  return <KpiStripSkeleton count={4} />;
}

export function DriftNotices({
  overview,
  providerName,
}: {
  overview: AccountingDriftOverview;
  providerName: string;
}) {
  const t = useT();

  if (!overview.checkError) {
    return null;
  }
  return (
    <Alert size="sm" variant="warning">
      <AlertTitle>{t("The last check could not read {0}", providerName)}</AlertTitle>
      <AlertDescription>{overview.checkError}</AlertDescription>
    </Alert>
  );
}
