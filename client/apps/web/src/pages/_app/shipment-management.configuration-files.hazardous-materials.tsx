import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { HazardousMaterialsPage } from "@/routes/hazardous-material/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute(
  "/_app/shipment-management/configuration-files/hazardous-materials",
)({
  beforeLoad: gates(requirePermission(Resource.HazardousMaterial)),
  component: HazardousMaterialsPage,
});
