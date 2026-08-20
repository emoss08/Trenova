import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useReportDashboard } from "@/hooks/use-reports";
import { usePermission } from "@/hooks/use-permission";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon } from "lucide-react";
import { useParams } from "react-router";
import { DashboardView } from "./_components/dashboard-view";

export function ReportDashboardPage() {
  const { dashboardId } = useParams<{ dashboardId: string }>();
  const dashboard = useReportDashboard(dashboardId);
  const { allowed: canEdit } = usePermission(Resource.Report, Operation.Update);

  if (dashboard.isLoading) {
    return (
      <div className="flex h-full flex-col gap-3 p-4">
        <Skeleton className="h-10" />
        <Skeleton className="flex-1" />
      </div>
    );
  }

  if (dashboard.isError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2">
        <CircleAlertIcon className="text-destructive size-8" />
        <p className="text-muted-foreground text-sm">
          {graphQLErrorMessage(dashboard.error, "Failed to load this dashboard")}
        </p>
      </div>
    );
  }

  if (!dashboard.data) return null;

  return <DashboardView key={dashboard.data.id} dashboard={dashboard.data} canEdit={canEdit} />;
}
