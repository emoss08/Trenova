import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { DeskConversationsPage } from "@/routes/admin/desk-conversations/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/desk-conversations")({
  beforeLoad: gates(requirePermission(Resource.DeskTranscript, Operation.Read)),
  component: DeskConversationsPage,
});
