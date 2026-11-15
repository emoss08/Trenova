import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BankReceiptQueuePage } from "@/routes/bank-receipt-queue/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reconciliation/work-queue")({
  beforeLoad: gates(requirePermission(Resource.BankReceiptWorkItem)),
  component: BankReceiptQueuePage,
});
