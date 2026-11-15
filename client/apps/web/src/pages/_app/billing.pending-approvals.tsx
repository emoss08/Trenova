import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InvoiceApprovalPage } from "@/routes/invoice-approval/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/pending-approvals")({
  beforeLoad: gates(requirePermission(Resource.Invoice)),
  component: InvoiceApprovalPage,
});
