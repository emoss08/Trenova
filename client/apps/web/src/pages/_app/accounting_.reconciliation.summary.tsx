import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ReconciliationSummaryPage } from "@/routes/reconciliation-summary/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reconciliation/summary")({
  beforeLoad: gates(requirePermission(Resource.BankReceipt)),
  component: ReconciliationSummaryPage,
});
