import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ShipmentTypesPage } from "@/routes/shipment-type/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute(
  "/_app/shipment-management/configuration-files/shipment-types",
)({
  beforeLoad: gates(requirePermission(Resource.ShipmentType)),
  component: ShipmentTypesPage,
});
