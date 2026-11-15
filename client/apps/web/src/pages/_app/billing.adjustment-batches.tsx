import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoiceAdjustmentBatchPage } from "@/routes/invoice-adjustment-batch/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/adjustment-batches")({
  beforeLoad: gates(requirePermission(Resource.Invoice)),
  component: InvoiceAdjustmentBatchPage,
});
