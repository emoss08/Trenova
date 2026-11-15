import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { APIKeysPage } from "@/routes/admin/api-keys/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/api-keys")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.APIKeys),
    requirePermission(Resource.APIKey, Operation.Read),
  ),
  component: APIKeysPage,
});
