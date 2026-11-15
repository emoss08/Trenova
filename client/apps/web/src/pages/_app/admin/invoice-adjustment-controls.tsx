import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoiceAdjustmentControlPage } from "@/routes/invoice-adjustment-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/invoice-adjustment-controls")({
  beforeLoad: gates(requirePermission(Resource.InvoiceAdjustmentControl, Operation.Read)),
  component: InvoiceAdjustmentControlPage,
});
