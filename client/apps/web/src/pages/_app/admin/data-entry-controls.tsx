import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DataEntryControlPage } from "@/routes/data-entry-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/data-entry-controls")({
  beforeLoad: gates(requirePermission(Resource.DataEntryControl, Operation.Read)),
  component: DataEntryControlPage,
});
