import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { ThreadSidebar } from "../thread-sidebar";

afterEach(cleanup);

function thread(overrides: Partial<AssistantThread>): AssistantThread {
  return {
    id: "athr_1",
    agentDefinitionId: "agtd_1",
    title: "Blocked invoices this week",
    createdAt: 1,
    lastMessageAt: 2,
    ...overrides,
  } as AssistantThread;
}

const agent = {
  id: "agtd_1",
  name: "Billing exceptions",
  description: "",
  template: "BillingException",
  icon: "receipt",
  accent: "amber",
  toolNames: [],
} as unknown as AgentDefinitionRow;

function renderSidebar() {
  return render(
    <ThreadSidebar
      threads={[thread({}), thread({ id: "athr_2", title: "Where is SEED-SHP-001?" })]}
      agentsById={new Map([[agent.id, agent]])}
      activeThreadId="athr_1"
      isLoading={false}
      canStart
      onSelect={() => {}}
      onStart={() => {}}
      onDelete={() => {}}
    />,
  );
}

/**
 * The history is a list of titles. Every row belongs to an agent, so a mark on
 * each one repeats the same handful of tiles down the whole column and competes
 * with the titles it is meant to be helping you read — the agent's name is
 * already on the second line, which is where that belongs.
 */
describe("ThreadSidebar rows", () => {
  it("carries no agent mark on a conversation row", () => {
    const { container } = renderSidebar();

    expect(container.querySelector(".agent-tile")).toBeNull();
  });

  it("still names the agent and the title", () => {
    renderSidebar();

    expect(screen.getByText("Blocked invoices this week")).toBeInTheDocument();
    expect(screen.getByText("Where is SEED-SHP-001?")).toBeInTheDocument();
    expect(screen.getAllByText(/Billing exceptions/).length).toBeGreaterThan(0);
  });
});
