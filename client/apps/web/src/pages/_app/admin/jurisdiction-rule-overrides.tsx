import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { JurisdictionRuleOverridesPage } from "@/routes/jurisdiction-rule-override/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/jurisdiction-rule-overrides")({
  beforeLoad: gates(requirePermission(Resource.JurisdictionRuleOverride)),
  component: JurisdictionRuleOverridesPage,
});
