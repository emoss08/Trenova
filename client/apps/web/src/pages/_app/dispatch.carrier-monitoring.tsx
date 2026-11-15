import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { CarrierMonitoringPage } from "@/routes/carrier-monitoring/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/carrier-monitoring")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.CarrierBrokerage),
    requirePermission(Resource.CarrierIntelligence),
  ),
  component: CarrierMonitoringPage,
});
