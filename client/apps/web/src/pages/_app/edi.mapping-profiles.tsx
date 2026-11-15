import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { EDIMappingProfilesPage } from "@/routes/edi/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/edi/mapping-profiles")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.Administration),
    requirePermission(Resource.EDI),
  ),
  component: EDIMappingProfilesPage,
});
