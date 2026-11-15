import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { LocationsPage } from "@/routes/location/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/locations")({
  beforeLoad: gates(requirePermission(Resource.Location)),
  component: LocationsPage,
});
