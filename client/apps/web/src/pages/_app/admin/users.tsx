import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { UsersPage } from "@/routes/admin/users/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/users")({
  beforeLoad: gates(requirePermission(Resource.User, Operation.Read)),
  component: UsersPage,
});
