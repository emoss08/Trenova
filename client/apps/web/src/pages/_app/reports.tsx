import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { ReportsPage, prefetch } from "@/routes/reports/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/reports")({
  beforeLoad: gates(requirePermission(Resource.Report, Operation.Read)),
  component: ReportsPage,
  loader: pageLoader(prefetch),
});
