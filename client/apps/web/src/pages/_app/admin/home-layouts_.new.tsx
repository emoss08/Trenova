import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { NewHomeLayoutPage } from "@/routes/admin/home-layouts/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/home-layouts_/new")({
  beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Create)),
  component: NewHomeLayoutPage,
});
