import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BalanceSheetPage } from "@/routes/balance-sheet/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reports/balance-sheet")({
  beforeLoad: gates(requirePermission(Resource.AccountingReport)),
  component: BalanceSheetPage,
});
