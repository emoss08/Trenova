import { createFileRoute } from "@tanstack/react-router";
import { anyOfGates, requirePermission } from "@/lib/route-permission";
import { DeskNeedsPage } from "@/routes/desk/needs-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

// Its Decisions lens is for whoever may read proposals; the
// rest of the page is the Watchtower's.
export const Route = createFileRoute("/_desk/desk/needs")({
  beforeLoad: gates(
    anyOfGates(
      requirePermission(Resource.Watchtower, Operation.Read),
      requirePermission(Resource.AgentProposal, Operation.Read),
    ),
  ),
  component: DeskNeedsPage,
});
