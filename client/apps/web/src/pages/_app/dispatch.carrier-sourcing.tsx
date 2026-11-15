import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { CarrierSourcingPage } from "@/routes/carrier-sourcing/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/carrier-sourcing")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.Brokerage),
    requirePlanFeature(PlanFeature.CarrierBrokerage),
    requirePermission(Resource.CarrierSourcing),
  ),
  component: CarrierSourcingPage,
});
