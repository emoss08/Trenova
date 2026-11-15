import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { AccountingDashboardPage } from "@/routes/accounting-dashboard/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting")({
  beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
  component: AccountingDashboardPage,
});
