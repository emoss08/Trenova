import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ReportDashboardPage } from "@/routes/reports/dashboards/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/reports_/dashboards/$dashboardId")({
  beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
  component: ReportDashboardPage,
});
