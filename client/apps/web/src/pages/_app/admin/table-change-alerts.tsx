import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { TableChangeAlertPage } from "@/routes/table-change-alert/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/table-change-alerts")({
  beforeLoad: gates(requirePermission(Resource.TableChangeAlert, Operation.Read)),
  component: TableChangeAlertPage,
});
