import type { ThreadHistory } from "@/components/assistant/thread-history";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import type { AssistantMessage } from "@/types/assistant";
import { useDeskStore } from "@/stores/desk-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Profiler } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { DeskHandoffMenu } from "../desk-handoff-menu";

afterEach(() => {
  cleanup();
  useDeskStore.setState({ liveHandoff: {} });
});

const THREAD_ID = "athr_1";

function agent(id: string, name: string, delegates: string[] = []): AgentChoice {
  return {
    id,
    name,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey: "",
    starters: [],
    delegates: delegates.map((delegate) => ({
      id: delegate,
      name: delegate,
      icon: "",
      accent: "",
      template: null,
    })),
  };
}

function message(
  sequence: number,
  overrides: Partial<AssistantMessage>,
): AssistantMessage {
  return {
    id: `amsg_${sequence}`,
    threadId: THREAD_ID,
    sequence,
    role: "Assistant",
    kind: "Message",
    content: "",
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: sequence,
    ...overrides,
  } as AssistantMessage;
}

function history(results: AssistantMessage[]): ThreadHistory {
  return {
    pages: [{ results, hasMore: false, total: results.length, limit: 50 }],
    pageParams: [undefined],
  };
}

function renderMenu(current: AgentChoice, agents: AgentChoice[], messages: AssistantMessage[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(queries.assistant.messages(THREAD_ID).queryKey, history(messages));
  let renders = 0;

  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Profiler
          id="handoff-menu"
          onRender={() => {
            renders += 1;
          }}
        >
          <DeskHandoffMenu threadId={THREAD_ID} agent={current} agents={agents} />
        </Profiler>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { client, renders: () => renders };
}

async function openMenu() {
  await userEvent.setup().click(screen.getByRole("button", { name: "Hand off to another agent" }));
  return within(screen.getByRole("menu")).getAllByRole("menuitem");
}

describe("DeskHandoffMenu", () => {
  const shipments = agent("shipments", "Shipment assistant", ["dispatch"]);
  const agents = [
    shipments,
    agent("dispatch", "Dispatch desk"),
    agent("exceptions", "Billing exceptions analyst"),
    agent("billing", "Billing assistant"),
  ];

  it("leads with the agent the conversation's agent named as holding the tool, marked", async () => {
    renderMenu(shipments, agents, [
      message(1, { role: "User", content: "Mark it ready to invoice" }),
      message(2, { role: "Assistant" }),
      message(3, { role: "Tool", toolName: "find_tools", handOffAgents: ["billing"] }),
      message(4, { role: "Assistant", content: "The billing assistant can do that." }),
    ]);

    const items = await openMenu();
    expect(items.map((item) => item.textContent)).toEqual([
      expect.stringContaining("Billing assistant"),
      expect.stringContaining("Dispatch desk"),
      expect.stringContaining("Billing exceptions analyst"),
    ]);
    expect(items[0]).toHaveTextContent("Holds what this conversation needs");
    expect(items[1]).not.toHaveTextContent("Holds what this conversation needs");
  });

  it("offers the agent the reply being written named, before the reply is saved", async () => {
    useDeskStore.getState().setLiveHandoff(THREAD_ID, { asked: true, agentIds: ["billing"] });
    renderMenu(shipments, agents, [
      message(1, { role: "User", content: "Earlier question" }),
      message(2, { role: "Assistant", content: "Earlier answer." }),
    ]);

    const items = await openMenu();
    expect(items[0]).toHaveTextContent("Billing assistant");
    expect(items[0]).toHaveTextContent("Holds what this conversation needs");
  });

  it("follows the reply as it names an agent, and renders again only when the suggestion changes", () => {
    const { client, renders } = renderMenu(shipments, agents, [
      message(1, { role: "User", content: "Mark it ready to invoice" }),
      message(2, { role: "Tool", toolName: "find_tools", handOffAgents: ["billing"] }),
      message(3, { role: "Assistant", content: "The billing assistant can do that." }),
    ]);
    const first = renders();

    // The reply that named it is still on screen: the same answer, from the live turn.
    act(() => {
      useDeskStore.getState().setLiveHandoff(THREAD_ID, { asked: true, agentIds: ["billing"] });
    });
    // It is saved and the live turn goes; the history now says the same.
    act(() => {
      client.setQueryData(
        queries.assistant.messages(THREAD_ID).queryKey,
        history([
          message(1, { role: "User", content: "Mark it ready to invoice" }),
          message(2, { role: "Tool", toolName: "find_tools", handOffAgents: ["billing"] }),
          message(3, { role: "Assistant", content: "The billing assistant can do that." }),
          message(4, { role: "Assistant", content: "Anything else?" }),
        ]),
      );
      useDeskStore.getState().setLiveHandoff(THREAD_ID, null);
    });
    expect(renders()).toBe(first);

    // The person asks again; the new reply names another agent.
    act(() => {
      useDeskStore.getState().setLiveHandoff(THREAD_ID, { asked: true, agentIds: [] });
    });
    expect(renders()).toBe(first + 1);
    act(() => {
      useDeskStore.getState().setLiveHandoff(THREAD_ID, { asked: true, agentIds: ["dispatch"] });
    });
    expect(renders()).toBe(first + 2);
  });

  it("keeps the usual order once the person has asked something else", async () => {
    renderMenu(shipments, agents, [
      message(1, { role: "Tool", toolName: "find_tools", handOffAgents: ["billing"] }),
      message(2, { role: "Assistant" }),
      message(3, { role: "User", content: "Never mind, where is load 1042?" }),
    ]);

    const items = await openMenu();
    expect(items[0]).toHaveTextContent("Dispatch desk");
    expect(screen.queryByText("Holds what this conversation needs")).not.toBeInTheDocument();
  });
});
