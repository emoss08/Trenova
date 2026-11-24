import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { defaultIftaPeriod, periodFromSearch, type IftaPeriodKey } from "@/lib/ifta-return";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { iftaReturn } from "@/lib/queries/ifta-return";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useQuery } from "@tanstack/react-query";
import { searchString } from "@trenova/shared/lib/router/search-codec";
import { lazy, useCallback, useMemo } from "react";
import { IftaReturnSkeleton } from "./_components/ifta-return-skeleton";
import { iftaPeriodQuery, iftaReturnForPeriodQuery } from "./_components/queries";

const IftaReturnWorkspace = lazy(() => import("./_components/ifta-return-workspace"));

const periodParsers = { year: searchString, quarter: searchString, return: searchString };

function nowUnix(): number {
  return Math.floor(Date.now() / 1000);
}

// The quarter is in the URL so a worksheet can be linked to, and an absent or
// unusable pair falls back to the most recently completed quarter — the one a
// preparer is working on.
export const prefetch: RoutePrefetch = ({ searchParams }) => {
  const search = searchParams;
  const period = periodFromSearch(
    search.get("year"),
    search.get("quarter"),
    defaultIftaPeriod(nowUnix()),
  );
  return [iftaReturnForPeriodQuery(period), iftaPeriodQuery(period)];
};

export function IftaReturnsPage() {
  const t = useT();

  const [search, setSearch] = useSearchState(periodParsers);
  const fallback = useMemo(() => defaultIftaPeriod(nowUnix()), []);
  // A link to one return (a Desk card's "Open IFTA return") names it by id;
  // its quarter is read from it until the person picks another.
  const linkedId = search.year ? null : search.return;
  const linked = useQuery({
    ...iftaReturn.card(linkedId ?? ""),
    enabled: Boolean(linkedId),
  });
  const period = linked.data
    ? { year: linked.data.year, quarter: linked.data.quarter }
    : periodFromSearch(search.year, search.quarter, fallback);
  const waitingForLinked = Boolean(linkedId) && linked.isPending;

  const onPeriodChange = useCallback(
    (next: IftaPeriodKey) => {
      void setSearch({ year: String(next.year), quarter: String(next.quarter) });
    },
    [setSearch],
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("IFTA returns"),
        description: t(
          "The quarterly fuel tax worksheet: every jurisdiction's miles and tax-paid gallons, the fleet MPG they are taxed through, and what the quarter owes or is owed. Draft figures move with the data until the return is finalized.",
        ),
      }}
    >
      {waitingForLinked ? (
        <IftaReturnSkeleton />
      ) : (
        <LazyComponent fallback={<IftaReturnSkeleton />}>
          <IftaReturnWorkspace period={period} onPeriodChange={onPeriodChange} />
        </LazyComponent>
      )}
    </PageLayout>
  );
}
