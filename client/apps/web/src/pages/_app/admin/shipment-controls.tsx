import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ShipmentControlPage } from "@/routes/shipment-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/shipment-controls")({
  beforeLoad: gates(requirePermission(Resource.ShipmentControl, Operation.Read)),
  component: ShipmentControlPage,
});
