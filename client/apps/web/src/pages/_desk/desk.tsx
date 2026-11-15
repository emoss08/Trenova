import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { DeskPage, prefetch } from "@/routes/desk/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_desk/desk")({
  beforeLoad: gates(requirePermission(Resource.Assistant, Operation.Read)),
  component: DeskPage,
  loader: pageLoader(prefetch),
});
