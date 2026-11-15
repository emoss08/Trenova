import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DistanceProfilesPage } from "@/routes/admin/distance-profiles/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/distance-profiles")({
  beforeLoad: gates(requirePermission(Resource.DistanceProfile)),
  component: DistanceProfilesPage,
});
