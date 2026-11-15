import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CommoditiesPage } from "@/routes/commodity/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/configuration-files/commodities")({
  beforeLoad: gates(requirePermission(Resource.Commodity)),
  component: CommoditiesPage,
});
