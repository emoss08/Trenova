import { createFileRoute } from "@tanstack/react-router";
import { requireCapability, requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { BenefitsPage, prefetch } from "@/routes/benefits/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { OrganizationCapability } from "@trenova/shared/types/organization-capability";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/hr/benefits")({
  beforeLoad: gates(
    requireCapability(OrganizationCapability.AssetOperations),
    requirePermission(Resource.BenefitPlan),
  ),
  component: BenefitsPage,
  loader: pageLoader(prefetch),
});
