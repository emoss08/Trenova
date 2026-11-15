import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CostControlPage } from "@/routes/cost-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/cost-control")({
  beforeLoad: gates(requirePermission(Resource.CostingControl)),
  component: CostControlPage,
});
