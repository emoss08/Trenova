import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { FactoringTab } from "../use-factoring-state";

export function FactoringSummaryStrip({
  active,
  onSelect,
}: {
  active: FactoringTab;
  onSelect: (tab: FactoringTab) => void;
}) {
  const t = useT();
  const { data, isLoading, isError, refetch } = useQuery(factoringQueries.summary());

  if (isLoading) {
    return <KpiStripSkeleton count={3} />;
  }

  if (isError || !data) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription className="flex items-center justify-between gap-3">
          <span>{t("The factoring summary could not be loaded.")}</span>
          <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
            {t("Retry")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <KpiStrip aria-label={t("Factoring summary")}>
      <KpiStripItem
        label={t("Ready")}
        value={formatCurrency(data.ready.amountMinor / 100)}
        sub={t("{0, plural, one {# invoice} other {# invoices}}", data.ready.count)}
        tone={data.ready.count > 0 ? "success" : undefined}
        active={active === "ready"}
        onClick={() => onSelect("ready")}
      />
      <KpiStripItem
        label={t("Not ready")}
        value={formatCurrency(data.notReady.amountMinor / 100)}
        sub={t("{0, plural, one {# invoice} other {# invoices}}", data.notReady.count)}
        tone={data.notReady.count > 0 ? "warning" : undefined}
        active={active === "not-ready"}
        onClick={() => onSelect("not-ready")}
      />
      <KpiStripItem
        label={t("Ready over a day")}
        value={formatCurrency(data.readyUnsubmitted.amountMinor / 100)}
        sub={t(
          "{0, plural, one {# invoice} other {# invoices}} not yet submitted",
          data.readyUnsubmitted.count,
        )}
        tone={data.readyUnsubmitted.count > 0 ? "danger" : undefined}
      />
    </KpiStrip>
  );
}
