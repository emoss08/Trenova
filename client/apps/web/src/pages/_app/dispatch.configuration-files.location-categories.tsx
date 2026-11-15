import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { LocationCategoriesPage } from "@/routes/location-category/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/configuration-files/location-categories")({
  beforeLoad: gates(requirePermission(Resource.LocationCategory)),
  component: LocationCategoriesPage,
});
