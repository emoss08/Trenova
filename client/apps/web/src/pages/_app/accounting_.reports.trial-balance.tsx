import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { TrialBalancePage } from "@/routes/trial-balance/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/reports/trial-balance")({
  beforeLoad: gates(requirePermission(Resource.AccountingReport)),
  component: TrialBalancePage,
});
