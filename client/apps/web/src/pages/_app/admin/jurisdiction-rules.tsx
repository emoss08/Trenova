import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { JurisdictionRulesPage } from "@/routes/jurisdiction-rule/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/jurisdiction-rules")({
  beforeLoad: gates(requirePermission(Resource.JurisdictionRule)),
  component: JurisdictionRulesPage,
});
