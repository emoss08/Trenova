import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ReportRunsPage } from "@/routes/reports/runs/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/reports_/runs")({
  beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
  component: ReportRunsPage,
});
