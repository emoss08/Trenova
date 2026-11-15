import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { AccountingSyncLedgerPage } from "@/routes/accounting-sync/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/sync")({
  beforeLoad: gates(requirePermission(Resource.AccountingSync)),
  component: AccountingSyncLedgerPage,
});
