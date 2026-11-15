import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DispatchControlPage } from "@/routes/dispatch-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/dispatch-controls")({
  beforeLoad: gates(requirePermission(Resource.DispatchControl, Operation.Read)),
  component: DispatchControlPage,
});
