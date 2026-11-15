import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { OrdersPage } from "@/routes/order/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/orders")({
  beforeLoad: gates(requirePermission(Resource.Order)),
  component: OrdersPage,
});
