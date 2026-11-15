import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { HomeLayoutsPage } from "@/routes/admin/home-layouts/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/home-layouts")({
  beforeLoad: gates(requirePermission(Resource.HomeLayoutPreset, Operation.Read)),
  component: HomeLayoutsPage,
});
