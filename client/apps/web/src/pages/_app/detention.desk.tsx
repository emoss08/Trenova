import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DetentionDeskPage } from "@/routes/detention-desk/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/detention/desk")({
  beforeLoad: gates(requirePermission(Resource.Detention)),
  component: DetentionDeskPage,
});
