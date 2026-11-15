import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { DetentionIntelligencePage, prefetch } from "@/routes/detention-intelligence/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/detention/intelligence")({
  beforeLoad: gates(requirePermission(Resource.Detention)),
  component: DetentionIntelligencePage,
  loader: pageLoader(prefetch),
});
