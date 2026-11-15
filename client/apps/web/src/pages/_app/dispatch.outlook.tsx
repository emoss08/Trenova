import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FreightOutlookPage, prefetch } from "@/routes/freight-outlook/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/dispatch/outlook")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePlanFeature(PlanFeature.DispatchIntelligence),
    requirePermission(Resource.ShipmentMove),
  ),
  component: FreightOutlookPage,
  loader: pageLoader(prefetch),
});
