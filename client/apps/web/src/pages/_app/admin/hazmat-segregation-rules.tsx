import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { HazmatSegregationRulesPage } from "@/routes/hazmat-segregation-rule/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/hazmat-segregation-rules")({
  beforeLoad: gates(requirePermission(Resource.HazmatSegregationRule)),
  component: HazmatSegregationRulesPage,
});
