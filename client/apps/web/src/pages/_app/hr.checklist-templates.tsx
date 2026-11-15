import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { WorkerChecklistTemplatesPage } from "@/routes/worker-checklist-template/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/checklist-templates")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.WorkerChecklistTemplate),
  ),
  component: WorkerChecklistTemplatesPage,
});
