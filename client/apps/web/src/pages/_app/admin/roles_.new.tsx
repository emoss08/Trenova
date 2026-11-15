import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RoleCreatePage } from "@/routes/admin/roles/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/roles_/new")({
  beforeLoad: gates(requirePermission(Resource.Role, Operation.Create)),
  component: RoleCreatePage,
});
