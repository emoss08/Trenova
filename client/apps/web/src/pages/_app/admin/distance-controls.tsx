import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DistanceControlsPage } from "@/routes/admin/distance-controls/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/distance-controls")({
  beforeLoad: gates(requirePermission(Resource.DistanceControl)),
  component: DistanceControlsPage,
});
