import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { EditHomeLayoutPage } from "@/routes/admin/home-layouts/[id]/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/home-layouts_/$id")({
  beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Update)),
  component: EditHomeLayoutPage,
});
