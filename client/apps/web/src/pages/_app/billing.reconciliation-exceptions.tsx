import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoiceReconciliationPage } from "@/routes/invoice-reconciliation/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/reconciliation-exceptions")({
  beforeLoad: gates(requirePermission(Resource.Invoice)),
  component: InvoiceReconciliationPage,
});
