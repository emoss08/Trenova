import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoiceRegisterPage } from "@/routes/invoice-register/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/invoice-register")({
  beforeLoad: gates(requirePermission(Resource.Invoice)),
  component: InvoiceRegisterPage,
});
