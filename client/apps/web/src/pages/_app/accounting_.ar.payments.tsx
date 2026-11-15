import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CustomerPaymentsPage } from "@/routes/customer-payments/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/ar/payments")({
  beforeLoad: gates(requirePermission(Resource.CustomerPayment)),
  component: CustomerPaymentsPage,
});
