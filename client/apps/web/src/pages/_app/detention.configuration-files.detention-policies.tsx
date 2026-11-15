import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DetentionPolicyPage } from "@/routes/detention-policy/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/detention/configuration-files/detention-policies")({
  beforeLoad: gates(requirePermission(Resource.DetentionPolicy)),
  component: DetentionPolicyPage,
});
