import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FleetSafetyPage, prefetch } from "@/routes/fleet-safety/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/fleet-safety")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.WorkerSafetyEvent),
  ),
  component: FleetSafetyPage,
  loader: pageLoader(prefetch),
});
