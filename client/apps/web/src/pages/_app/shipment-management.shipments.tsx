import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { ShipmentsPage, prefetch } from "@/routes/shipment/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/shipment-management/shipments")({
  beforeLoad: gates(requirePermission(Resource.Shipment)),
  component: ShipmentsPage,
  loader: pageLoader(prefetch),
});
