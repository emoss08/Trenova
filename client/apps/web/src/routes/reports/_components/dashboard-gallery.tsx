import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportDashboards } from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { parseDashboardLayout } from "@/types/report";
import { CircleAlertIcon, LayoutDashboardIcon, PlusIcon } from "lucide-react";
import { useMemo } from "react";
import { useNavigate } from "react-router";
import {
  CategoryTile,
  ReportCard,
  ReportGridEmpty,
  ReportGridEmptyState,
} from "./report-card-chrome";
import { useCreateDashboardAction } from "./use-create-dashboard-action";
import { compareReportsBySort, type ReportSortOrder } from "../reports-page-state";

type DashboardGalleryProps = {
  search: string;
  sortBy: ReportSortOrder;
  onClearFilters: () => void;
};

export function DashboardGallery({ search, sortBy, onClearFilters }: DashboardGalleryProps) {
  const t = useT();

  const navigate = useNavigate();
  const dashboards = useReportDashboards();
  const createDashboard = useCreateDashboardAction();
  const { allowed: canCreate } = usePermission(Resource.Report, Operation.Create);

  const visible = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return (dashboards.data ?? [])
      .filter(
        (dashboard) =>
          !needle ||
          dashboard.name.toLowerCase().includes(needle) ||
          dashboard.description.toLowerCase().includes(needle),
      )
      .sort(compareReportsBySort(sortBy));
  }, [dashboards.data, search, sortBy]);

  if (dashboards.isLoading) {
    return (
      <div className="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-28 rounded-lg" />
        ))}
      </div>
    );
  }

  // Surface the failure instead of the empty state — an unmigrated database
  // otherwise looks identical to "you have no dashboards yet".
  if (dashboards.isError) {
    return (
      <div className="grid p-4">
        <ReportGridEmptyState
          icon={CircleAlertIcon}
          title={t("Dashboards could not be loaded")}
          description={graphQLErrorMessage(dashboards.error, "Please try again")}
        />
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {visible.length === 0 ? (
          <ReportGridEmpty
            variant="dashboards"
            title={search.trim() ? "Nothing matches" : "No dashboards yet"}
            description={
              search.trim()
                ? "No dashboard fits that search. Clear it to see every dashboard."
                : "A dashboard puts saved reports side by side under one set of filters. Create one and lay your reports out on it."
            }
            onClearFilters={search.trim() ? onClearFilters : undefined}
            action={
              canCreate ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={createDashboard.create}
                  disabled={createDashboard.isPending}
                >
                  <PlusIcon className="size-3.5" />
                  {t("New dashboard")}
                </Button>
              ) : undefined
            }
          />
        ) : (
          visible.map((dashboard, index) => {
            const layout = parseDashboardLayout(dashboard.layout);
            return (
              <ReportCard
                key={dashboard.id}
                index={index}
                onClick={() => void navigate(`/reports/dashboards/${dashboard.id}`)}
              >
                <div className="flex items-start gap-3">
                  <CategoryTile category={dashboard.category || "dashboard"} />
                  <div className="min-w-0 flex-1">
                    <h3 className="truncate text-sm font-medium">{dashboard.name}</h3>
                    <p className="text-muted-foreground mt-0.5 line-clamp-2 min-h-8 text-xs">
                      {dashboard.description || t("No description")}
                    </p>
                  </div>
                </div>
                <div className="border-border/60 text-2xs text-muted-foreground mt-3 flex items-center gap-2 border-t pt-3">
                  <LayoutDashboardIcon className="size-3.5" />
                  <span className="tabular-nums">
                    {t("{0, plural, one {# tile} other {# tiles}}", layout.tiles.length)}
                  </span>
                  {(layout.parameters?.length ?? 0) > 0 && (
                    <span className="tabular-nums">
                      {t(
                        "· {0, plural, one {# filter} other {# filters}}",
                        layout.parameters?.length,
                      )}
                    </span>
                  )}
                  <div className="flex-1" />
                  <span className="capitalize">{dashboard.visibility}</span>
                </div>
              </ReportCard>
            );
          })
        )}
      </div>
    </div>
  );
}
