import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ReportExplorePage } from "@/routes/reports/explore/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/reports_/explore")({
  beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
  component: ReportExplorePage,
});
