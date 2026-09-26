import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import {
  tableFilterSearchParamsParser,
  tablePaginationSearchParamsParser,
} from "@/hooks/data-table/use-data-table-state";
import { useAccountingInboundLabels } from "@/hooks/use-accounting-inbound-labels";
import { accountingSetupPath } from "@/lib/accounting-sync";
import type { AccountingInboundOverview } from "@/lib/graphql/accounting-inbound";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { useQueryStates } from "nuqs";
import { Link } from "react-router";
import { activeInboundBucket, inboundBucketFilter, type InboundBucket } from "./inbound-filters";

export function InboundSummary({ overview }: { overview: AccountingInboundOverview }) {
  const t = useT();
  const [filters, setFilters] = useQueryStates({
    ...tableFilterSearchParamsParser,
    ...tablePaginationSearchParamsParser,
  });
  const current = activeInboundBucket(filters.fieldFilters);
  const { summary } = overview;

  const select = (bucket: InboundBucket) => {
    void setFilters({
      fieldFilters: current === bucket ? [] : inboundBucketFilter(bucket),
      pageIndex: 1,
    });
  };

  return (
    <KpiStrip>
      <KpiStripItem
        label={t("Waiting for you")}
        value={summary.proposed}
        tone={summary.proposed > 0 ? "warning" : undefined}
        active={current === "proposed"}
        onClick={() => select("proposed")}
      />
      <KpiStripItem
        label={t("Just read")}
        value={summary.detected}
        sub={t("checked in a few minutes")}
        active={current === "detected"}
        onClick={() => select("detected")}
      />
      <KpiStripItem
        label={t("Applied")}
        value={summary.appliedSince}
        sub={t("last 7 days")}
        active={current === "applied"}
        onClick={() => select("applied")}
      />
      <KpiStripItem
        label={t("Ignored")}
        value={summary.ignoredSince}
        sub={t("last 7 days")}
        active={current === "ignored"}
        onClick={() => select("ignored")}
      />
    </KpiStrip>
  );
}

export function InboundSummarySkeleton() {
  return <KpiStripSkeleton count={4} />;
}

export function InboundNotices({
  overview,
  system,
  providerName,
}: {
  overview: AccountingInboundOverview;
  system: AccountingSystem;
  providerName: string;
}) {
  const t = useT();
  const labels = useAccountingInboundLabels();

  return (
    <>
      {overview.changesError ? (
        <Alert size="sm" variant="warning">
          <AlertTitle>
            {t("Trenova could not read the latest changes from {0}", providerName)}
          </AlertTitle>
          <AlertDescription>{overview.changesError}</AlertDescription>
        </Alert>
      ) : null}
      <Alert size="sm" variant={overview.policy === "Off" ? "warning" : "info"}>
        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
          <span>
            {overview.policy === "Off"
              ? t(
                  "Payments recorded in {0} are left out of Trenova. Turn them on in the integration settings.",
                  providerName,
                )
              : t(
                  "Payments recorded in {0}: {1}. Last read {2}.",
                  providerName,
                  labels.policy[overview.policy].toLowerCase(),
                  overview.changesReadAt
                    ? formatUnixDateTimeShort(overview.changesReadAt)
                    : t("not yet"),
                )}
          </span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            render={<Link to={accountingSetupPath(system)} />}
          >
            {t("Integration settings")}
          </Button>
        </AlertDescription>
      </Alert>
    </>
  );
}
