import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { DispatchConsolePage, prefetch } from "@/routes/dispatch-console/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/console")({
  beforeLoad: gates(requirePermission(Resource.ShipmentMove)),
  component: DispatchConsolePage,
  loader: pageLoader(prefetch),
});
