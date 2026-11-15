import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { AgentControlPage } from "@/routes/agent-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/agent-control")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.AgentAutomation),
    requirePermission(Resource.AgentControl, Operation.Read),
  ),
  component: AgentControlPage,
});
