import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { WorkerCreatePage } from "@/routes/worker/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/workers_/new")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.Worker, Operation.Create),
  ),
  component: WorkerCreatePage,
});
