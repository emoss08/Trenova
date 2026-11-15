import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BankReceiptBatchDetailPage } from "@/routes/bank-receipt-batch/detail-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reconciliation/import-batches_/$batchId")({
  beforeLoad: gates(requirePermission(Resource.BankReceipt)),
  component: BankReceiptBatchDetailPage,
});
