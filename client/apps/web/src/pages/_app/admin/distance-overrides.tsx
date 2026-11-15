import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DistanceOverridesPage } from "@/routes/distance-override/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/distance-overrides")({
  beforeLoad: gates(requirePermission(Resource.DistanceOverride)),
  component: DistanceOverridesPage,
});
