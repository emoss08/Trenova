import { NarrativeSegments } from "@/components/narrative-segments";
import type { FactoringOverviewSegment } from "@/lib/graphql/factoring-company-page";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FactoringOverviewTarget } from "@trenova/graphql/generated/graphql";
import type { FactoringCompanyTab } from "../use-factoring-company-state";
import { FactoringAttentionList } from "./factoring-attention-list";
import { FactoringCompanyOverviewSkeleton } from "./factoring-company-skeleton";
import { FactoringMoneyPath } from "./factoring-money-path";
import { FactoringPaceChart } from "./factoring-pace-chart";
import { FactoringSetupSteps } from "./factoring-setup-steps";
import { FactoringTermsSummary } from "./factoring-terms-summary";

const TARGET_TABS: Record<FactoringOverviewTarget, FactoringCompanyTab> = {
  Delivery: "delivery",
  Customers: "customers",
  Ledger: "ledger",
  Agreement: "agreement",
};

/**
 * The factor at a glance: Nova's summary, where its invoices are on their way to cash (or what
 * stands between a new factor and its first schedule), what needs someone, the terms in force and
 * how fast it funds.
 */
export default function FactoringCompanyOverview({
  factoringCompanyId,
  onTab,
}: {
  factoringCompanyId: string;
  onTab: (tab: FactoringCompanyTab) => void;
}) {
  const t = useT();
  const { data, error, isLoading, refetch } = useQuery(
    factoringQueries.overview(factoringCompanyId),
  );

  if (isLoading) return <FactoringCompanyOverviewSkeleton />;
  if (error || !data) {
    return (
      <ErrorState
        layout="sheet"
        error={error}
        onRetry={() => void refetch()}
        title={t("The overview could not be loaded")}
      />
    );
  }

  const segments = data.summary.map((segment) => ({
    ...segment,
    targetId: segment.target ?? null,
  }));
  const settingUp =
    data.funded.count === 0 &&
    data.submitted.count === 0 &&
    data.released.count === 0 &&
    data.setup.done < 4;

  return (
    <div className="flex flex-col gap-5">
      <section aria-label={t("Summary")} className="flex flex-col gap-2">
        <span className="flex items-center gap-2 text-xs text-muted-foreground">
          <NovaBadge size="sm" />
          <span className="font-medium text-foreground">{t("Nova")}</span>
          {t("Factor summary")}
        </span>
        <p className="max-w-5xl text-lg leading-relaxed text-pretty text-muted-foreground">
          <NarrativeSegments<FactoringOverviewSegment & { targetId: string | null }>
            segments={segments}
            onTarget={(segment) => {
              if (segment.target) onTab(TARGET_TABS[segment.target]);
            }}
          />
        </p>
      </section>
      {settingUp ? (
        <FactoringSetupSteps overview={data} onTab={onTab} />
      ) : (
        <FactoringMoneyPath overview={data} factoringCompanyId={factoringCompanyId} onTab={onTab} />
      )}
      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
        <FactoringAttentionList
          overview={data}
          factoringCompanyId={factoringCompanyId}
          onTab={onTab}
        />
        <div className="flex min-w-0 flex-col gap-4">
          <FactoringTermsSummary agreement={data.agreement} onTab={onTab} />
          <FactoringPaceChart overview={data} />
        </div>
      </div>
    </div>
  );
}
