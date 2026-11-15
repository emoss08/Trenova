import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { InsightsPage } from "@/routes/insights/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/insights")({
  beforeLoad: gates(requirePermission(Resource.Insight, Operation.Read)),
  component: InsightsPage,
});
