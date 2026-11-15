import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CustomFieldDefinitionsPage } from "@/routes/admin/custom-fields/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/custom-fields")({
  beforeLoad: gates(requirePermission(Resource.CustomFieldDefinition, Operation.Read)),
  component: CustomFieldDefinitionsPage,
});
