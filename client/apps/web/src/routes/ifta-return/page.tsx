import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { defaultIftaPeriod, periodFromSearch, type IftaPeriodKey } from "@/lib/ifta-return";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { parseAsString, useQueryStates } from "nuqs";
import { lazy, useCallback, useMemo } from "react";
import { IftaReturnSkeleton } from "./_components/ifta-return-skeleton";
import { iftaPeriodQuery, iftaReturnForPeriodQuery } from "./_components/queries";

const IftaReturnWorkspace = lazy(() => import("./_components/ifta-return-workspace"));

const periodParsers = { year: parseAsString, quarter: parseAsString };

function nowUnix(): number {
  return Math.floor(Date.now() / 1000);
}

// The quarter is in the URL so a worksheet can be linked to, and an absent or
// unusable pair falls back to the most recently completed quarter — the one a
// preparer is working on.
export const prefetch: RoutePrefetch = ({ request }) => {
  const search = new URL(request.url).searchParams;
  const period = periodFromSearch(
    search.get("year"),
    search.get("quarter"),
    defaultIftaPeriod(nowUnix()),
  );
  return [iftaReturnForPeriodQuery(period), iftaPeriodQuery(period)];
};

export function IftaReturnsPage() {
  const t = useT();

  const [search, setSearch] = useQueryStates(periodParsers);
  const fallback = useMemo(() => defaultIftaPeriod(nowUnix()), []);
  const period = periodFromSearch(search.year, search.quarter, fallback);

  const onPeriodChange = useCallback(
    (next: IftaPeriodKey) => {
      void setSearch({ year: String(next.year), quarter: String(next.quarter) });
    },
    [setSearch],
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("IFTA Returns"),
        description: t(
          "The quarterly fuel tax worksheet: every jurisdiction's miles and tax-paid gallons, the fleet MPG they are taxed through, and what the quarter owes or is owed. Draft figures move with the data until the return is finalized.",
        ),
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<IftaReturnSkeleton />}>
          <IftaReturnWorkspace period={period} onPeriodChange={onPeriodChange} />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
