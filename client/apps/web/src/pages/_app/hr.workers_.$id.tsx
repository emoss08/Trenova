import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { WorkerPage, prefetch } from "@/routes/worker/[id]/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/workers_/$id")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.Worker),
  ),
  component: WorkerPage,
  loader: pageLoader(prefetch),
});
