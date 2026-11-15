import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RolesPage } from "@/routes/admin/roles/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/roles")({
  beforeLoad: gates(requirePermission(Resource.Role, Operation.Read)),
  component: RolesPage,
});
