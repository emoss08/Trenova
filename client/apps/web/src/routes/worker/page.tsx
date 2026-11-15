import { readSearchState } from "@trenova/shared/hooks/use-search-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { PlanFeature } from "@/lib/plan-feature";
import { tableConfiguration } from "@/lib/queries/table-configuration";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { planFeatureOf, usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
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

const loadWorkersSearch = (search: URLSearchParams) =>
  readSearchState(
    {
      [WORKERS_PAGE_TAB_PARAM]: workersPageTabParser,
      ...ptoOverviewFiltersSearchParamsParser,
      ...ptoViewTypeSearchParamsParser,
    },
    search,
  );

// The approved-PTO chart sits above the tabs and fires for the month in the URL (or the
// parser's default month) whenever the chart view is showing; the calendar view mounts
// a different component with its own queries. Below it, only the active tab mounts:
// the roster's saved default view, or the PTO tab's balance summary and — for a manager
// — its liability report. The rows of either table are paginated and left to the table.
// Paid time off is the HR add-on's: without it none of that is drawn or read.
export const prefetch: RoutePrefetch = ({ searchParams }) => {
  const { pageTab, ptoOverviewFilters, viewType } = loadWorkersSearch(searchParams);
  const timezone = useAuthStore.getState().user?.timezone;
  const permissions = usePermissionStore.getState();
  const timeOffOnPlan = planFeatureOf(permissions.manifest, PlanFeature.WorkforceTimeOff).available;
  const list: RoutePrefetchQuery[] = [];

  if (!timeOffOnPlan) {
    return [{ ...tableConfiguration.default(WORKER_TABLE_NAME), staleTime: Infinity }];
  }

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
    if (permissions.hasPermission(Resource.WorkerPTO, Operation.Manage)) {
      list.push(ptoLiabilityReportQuery());
    }
  } else {
    list.push({ ...tableConfiguration.default(WORKER_TABLE_NAME), staleTime: Infinity });
  }

  return list;
};

export function WorkersPage() {
  const t = useT();
  const timeOffOnPlan = usePlanFeature(PlanFeature.WorkforceTimeOff).available;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Workers"),
        description: t("Manage and track workers along with their compliance and paid time off"),
      }}
    >
      {timeOffOnPlan ? <PTOContent /> : null}
      <WorkersContent timeOffOnPlan={timeOffOnPlan} />
    </PageLayout>
  );
}
