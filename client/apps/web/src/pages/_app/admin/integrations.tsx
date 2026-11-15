import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { IntegrationsPage } from "@/routes/admin/integrations/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/integrations")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.Administration),
    requirePermission(Resource.Integration, Operation.Read),
  ),
  component: IntegrationsPage,
});
