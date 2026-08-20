import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportDashboards } from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { parseDashboardLayout } from "@/types/report";
import { CircleAlertIcon, LayoutDashboardIcon, LayoutGridIcon, PlusIcon } from "lucide-react";
import { useMemo } from "react";
import { useNavigate } from "react-router";
import { CategoryTile, ReportCard, ReportGridEmptyState } from "./report-card-chrome";
import { useCreateDashboardAction } from "./use-create-dashboard-action";
import { compareReportsBySort, type ReportSortOrder } from "../reports-page-state";

type DashboardGalleryProps = {
  search: string;
  sortBy: ReportSortOrder;
};

export function DashboardGallery({ search, sortBy }: DashboardGalleryProps) {
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
          title="Dashboards could not be loaded"
          description={graphQLErrorMessage(dashboards.error, "Please try again")}
        />
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {visible.length === 0 ? (
          <ReportGridEmptyState
            icon={LayoutGridIcon}
            title="No dashboards yet"
            description="A dashboard puts saved reports side by side under one set of filters."
            action={
              canCreate ? (
                <Button
                  size="sm"
                  onClick={createDashboard.create}
                  disabled={createDashboard.isPending}
                >
                  <PlusIcon className="size-3.5" />
                  New Dashboard
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
                      {dashboard.description || "No description"}
                    </p>
                  </div>
                </div>
                <div className="border-border/60 text-2xs text-muted-foreground mt-3 flex items-center gap-2 border-t pt-3">
                  <LayoutDashboardIcon className="size-3.5" />
                  <span className="tabular-nums">
                    {layout.tiles.length} tile{layout.tiles.length === 1 ? "" : "s"}
                  </span>
                  {(layout.parameters?.length ?? 0) > 0 && (
                    <span className="tabular-nums">
                      · {layout.parameters?.length} filter
                      {layout.parameters?.length === 1 ? "" : "s"}
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
