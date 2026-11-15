import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DeskHandledPage } from "@/routes/desk/handled-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_desk/desk/handled/$id")({
  beforeLoad: gates(requirePermission(Resource.AgentProposal, Operation.Read)),
  component: DeskHandledPage,
});
