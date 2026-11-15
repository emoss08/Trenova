import { createFileRoute } from "@tanstack/react-router";
import { PlanFeature } from "@/lib/plan-feature";
import { requirePlanFeature } from "@/lib/plan-feature-gate";
import { requirePermission } from "@/lib/route-permission";
import { EmailLogsPage } from "@/routes/organization/email-logs/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/organization/email-logs")({
  beforeLoad: gates(
    requirePlanFeature(PlanFeature.Administration),
    requirePermission(Resource.EmailLog),
  ),
  component: EmailLogsPage,
});
