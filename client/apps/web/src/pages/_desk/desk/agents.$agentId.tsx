import { createFileRoute } from "@tanstack/react-router";
import { DeskAgentCapabilitiesPage } from "@/routes/desk/agent-page";

export const Route = createFileRoute("/_desk/desk/agents/$agentId")({
  component: DeskAgentCapabilitiesPage,
});
