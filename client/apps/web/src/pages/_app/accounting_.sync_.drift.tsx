import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { AccountingDriftPage } from "@/routes/accounting-drift/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/sync_/drift")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.Accounting),
    requirePermission(Resource.AccountingReconciliation),
  ),
  component: AccountingDriftPage,
});
