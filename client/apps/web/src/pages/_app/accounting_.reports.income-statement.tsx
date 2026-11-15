import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { IncomeStatementPage } from "@/routes/income-statement/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reports/income-statement")({
  beforeLoad: gates(requirePermission(Resource.AccountingReport)),
  component: IncomeStatementPage,
});
