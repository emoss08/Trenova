import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoicesPage } from "@/routes/invoice/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/invoices")({
  beforeLoad: gates(requirePermission(Resource.Invoice)),
  component: InvoicesPage,
});
