import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BankReceiptPage } from "@/routes/bank-receipt/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reconciliation/bank-receipts")({
  beforeLoad: gates(requirePermission(Resource.BankReceipt)),
  component: BankReceiptPage,
});
