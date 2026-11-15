import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BankReceiptBatchPage } from "@/routes/bank-receipt-batch/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reconciliation/import-batches")({
  beforeLoad: gates(requirePermission(Resource.BankReceipt)),
  component: BankReceiptBatchPage,
});
