import { useParams } from "@tanstack/react-router";
import { DeskAgentPage } from "./_components/agent/desk-agent-page";

/** What an agent can do, opened from its name or the shield in the top bar. */
export function DeskAgentCapabilitiesPage() {
  const { agentId = "" } = useParams({ strict: false });

  return <DeskAgentPage key={agentId} agentId={agentId} />;
}
