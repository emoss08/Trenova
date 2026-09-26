import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import {
  tableFilterSearchParamsParser,
  tablePaginationSearchParamsParser,
} from "@/hooks/data-table/use-data-table-state";
import type { JournalReviewSummary } from "@/lib/graphql/journal-review";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { useQueryStates } from "nuqs";
import {
  activeJournalReviewBucket,
  journalReviewBucketFilter,
  type JournalReviewBucket,
} from "./review-filters";

export function JournalReviewSummaryStrip({ summary }: { summary: JournalReviewSummary }) {
  const t = useT();
  const [filters, setFilters] = useQueryStates({
    ...tableFilterSearchParamsParser,
    ...tablePaginationSearchParamsParser,
  });
  const current = activeJournalReviewBucket(filters.fieldFilters);

  const select = (bucket: JournalReviewBucket) => {
    void setFilters({
      fieldFilters: current === bucket ? [] : journalReviewBucketFilter(bucket),
      pageIndex: 1,
    });
  };

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("Awaiting approval")}
        value={summary.awaitingApproval}
        tone={summary.awaitingApproval > 0 ? "warning" : undefined}
        active={current === "awaiting"}
        onClick={() => select("awaiting")}
      />
      <KpiStripItem
        label={t("Ready to post")}
        value={summary.readyToPost}
        active={current === "ready"}
        onClick={() => select("ready")}
      />
      <KpiStripItem
        label={t("Oldest waiting")}
        value={formatUnixDateMedium(summary.oldestAccountingDate, { fallback: "—" })}
        sub={t("accounting date")}
      />
    </KpiStrip>
  );
}

export function JournalReviewSummarySkeleton() {
  return <KpiStripSkeleton count={3} />;
}
