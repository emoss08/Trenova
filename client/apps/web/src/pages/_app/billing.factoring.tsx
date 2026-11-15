import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { FactoringPage } from "@/routes/factoring/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/factoring")({
  beforeLoad: gates(requirePlanFeature(PlanFeature.Factoring), requirePermission(Resource.Invoice)),
  component: FactoringPage,
});
