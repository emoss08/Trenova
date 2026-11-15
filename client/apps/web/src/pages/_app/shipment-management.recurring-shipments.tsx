import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RecurringShipmentsPage } from "@/routes/recurring-shipment/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/recurring-shipments")({
  beforeLoad: gates(requirePermission(Resource.RecurringShipment)),
  component: RecurringShipmentsPage,
});
