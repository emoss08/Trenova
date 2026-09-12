import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createLoader } from "nuqs";
import WorkersContent, {
  WORKERS_PAGE_TAB_PARAM,
  workersPageTabParser,
} from "./_components/page-content";
import { PTOContent } from "./_components/pto-content";
import {
  approvedPtoChartQuery,
  requestedPtoCountQuery,
  type ApprovedPTOAnalyticsParams,
} from "./_components/pto/approved/chart/use-approved-pto-analytics";
import { ptoBalanceSummaryQuery, ptoLiabilityReportQuery } from "./_components/pto/pto-queries";
import {
  ptoOverviewFiltersSearchParamsParser,
  ptoViewTypeSearchParamsParser,
} from "./_components/pto/use-pto-state";

const WORKER_TABLE_NAME = "Worker";

const loadWorkersSearch = createLoader({
  [WORKERS_PAGE_TAB_PARAM]: workersPageTabParser,
  ...ptoOverviewFiltersSearchParamsParser,
  ...ptoViewTypeSearchParamsParser,
});

// The approved-PTO chart sits above the tabs and fires for the month in the URL (or the
// parser's default month) whenever the chart view is showing; the calendar view mounts
// a different component with its own queries. Below it, only the active tab mounts:
// the roster's saved default view, or the PTO tab's balance summary and — for a manager
// — its liability report. The rows of either table are paginated and left to the table.
export const prefetch: RoutePrefetch = ({ request }) => {
  const { pageTab, ptoOverviewFilters, viewType } = loadWorkersSearch(request);
  const timezone = useAuthStore.getState().user?.timezone;
  const list: RoutePrefetchQuery[] = [];

  if (viewType === "chart") {
    const params: ApprovedPTOAnalyticsParams = {
      startDate: ptoOverviewFilters.startDate,
      endDate: ptoOverviewFilters.endDate,
      type: ptoOverviewFilters.type ?? undefined,
      workerId: ptoOverviewFilters.workerId ?? undefined,
      fleetCodeId: ptoOverviewFilters.fleetCodeId ?? undefined,
    };
    list.push(approvedPtoChartQuery(params, timezone), requestedPtoCountQuery(params, timezone));
  }

  if (pageTab === "pto") {
    list.push(ptoBalanceSummaryQuery());
    if (usePermissionStore.getState().hasPermission(Resource.WorkerPTO, Operation.Manage)) {
      list.push(ptoLiabilityReportQuery());
    }
  } else {
    list.push({ ...queries.tableConfiguration.default(WORKER_TABLE_NAME), staleTime: Infinity });
  }

  return list;
};

export function WorkersPage() {
  const t = useT();

  return (
    <PageLayout
      className="gap-y-2"
      pageHeaderProps={{
        title: t("Workers"),
        description: t("Manage and track workers along with their compliance and paid time off"),
      }}
    >
      <PTOContent />
      <WorkersContent />
    </PageLayout>
  );
}
