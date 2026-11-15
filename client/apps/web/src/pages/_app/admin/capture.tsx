import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { CaptureAdminPage } from "@/routes/admin/capture/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/capture")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.DocumentIntelligence),
    requirePermission(Resource.CaptureProfile, Operation.Read),
  ),
  component: CaptureAdminPage,
});
