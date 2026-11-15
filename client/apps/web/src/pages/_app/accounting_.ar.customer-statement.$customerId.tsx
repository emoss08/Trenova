import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CustomerStatementPage } from "@/routes/customer-statement/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/ar/customer-statement/$customerId")({
  beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
  component: CustomerStatementPage,
});
