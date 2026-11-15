import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RoleEditPage } from "@/routes/admin/roles/[id]/edit/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/roles_/$id/edit")({
  beforeLoad: gates(requirePermission(Resource.Role, Operation.Update)),
  component: RoleEditPage,
});
