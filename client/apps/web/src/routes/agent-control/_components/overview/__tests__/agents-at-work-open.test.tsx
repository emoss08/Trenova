import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AgentsAtWork } from "../agents-at-work";

function agent(id: string, name: string, patch: Partial<AgentDefinitionRow> = {}) {
  return {
    id,
    name,
    icon: "bot",
    accent: "indigo",
    enabled: true,
    shadowMode: false,
    pendingProposals: 0,
    ...patch,
  } as AgentDefinitionRow;
}

function renderAgents(agents: AgentDefinitionRow[], runs: WorkingRun[]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  client.setQueryData(queries.assistant.agents(false).queryKey, agents);
  client.setQueryData(queries.aiControl.workingRuns().queryKey, runs);
  const onOpenAgent = vi.fn();
  const onOpenAgents = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <AgentsAtWork idleReason={null} onOpenAgent={onOpenAgent} onOpenAgents={onOpenAgents} />
    </QueryClientProvider>,
  );
  return { onOpenAgent, onOpenAgents };
}

describe("AgentsAtWork", () => {
  it("opens the agent whose tile is clicked, not the roster", () => {
    const { onOpenAgent, onOpenAgents } = renderAgents(
      [agent("agdef_a", "Dispatch desk"), agent("agdef_b", "Billing", { enabled: false })],
      [],
    );

    fireEvent.click(screen.getByTitle("Billing · off"));
    expect(onOpenAgent).toHaveBeenCalledWith("agdef_b");
    expect(onOpenAgents).not.toHaveBeenCalled();
  });

  it("opens the agent a working row belongs to", () => {
    const { onOpenAgent } = renderAgents(
      [agent("agdef_a", "Dispatch desk"), agent("agdef_b", "Billing")],
      [
        {
          id: "run_1",
          agentDefinitionId: "agdef_b",
          status: "Diagnosing",
          summary: "Reading invoices",
          startedAt: 1,
        } as WorkingRun,
      ],
    );

    fireEvent.click(screen.getByText("Reading invoices…").closest("button")!);
    expect(onOpenAgent).toHaveBeenCalledWith("agdef_b");
  });

  it("keeps the roster link for the whole list", () => {
    const { onOpenAgent, onOpenAgents } = renderAgents([agent("agdef_a", "Dispatch desk")], []);

    fireEvent.click(screen.getByRole("button", { name: /Roster/ }));
    expect(onOpenAgents).toHaveBeenCalledOnce();
    expect(onOpenAgent).not.toHaveBeenCalled();
  });
});
