import type { AgentChoice, AgentRecommendation } from "@/lib/graphql/agent-definition";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskHome, type DeskHomeProps } from "../desk-home";

const state = vi.hoisted(() => ({
  agent: null as unknown,
  recommend: (() => Promise.resolve(null)) as (
    text: string,
    currentAgentId: string,
  ) => Promise<unknown>,
  asked: [] as { text: string; currentAgentId: string }[],
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/lib/queries/briefing", () => ({
  briefing: {
    today: () => ({ queryKey: ["briefing-today"], queryFn: () => null }),
  },
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
    providers: () => ({ queryKey: ["providers"], queryFn: () => [] }),
    agentRecommendation: (text: string, currentAgentId: string | null) => ({
      queryKey: ["agent-recommendation", text, currentAgentId ?? ""],
      queryFn: () => {
        state.asked.push({ text, currentAgentId: currentAgentId ?? "" });
        return state.recommend(text, currentAgentId ?? "");
      },
    }),
  },
}));

vi.mock("../decisions/use-pending-decisions", () => ({
  usePendingDecisionSummary: () => ({
    isPending: false,
    data: { total: 0, byAgent: [], oldestAt: null },
  }),
}));

vi.mock("@/hooks/use-attention", () => ({
  useAttentionSummary: () => ({ data: undefined }),
}));

vi.mock("@/components/assistant/use-askable-agent", async () => {
  const { useState } = await import("react");

  return {
    useAskableAgent: () => {
      const [chosen, setChosen] = useState<unknown>(null);

      return {
        agent: chosen ?? state.agent,
        choose: setChosen,
        recency: { ids: [], lastUsedAt: new Map() },
        choices: {
          isLoading: false,
          isError: false,
          recent: [],
          items: [],
          refetch: () => undefined,
        },
        noneAvailable: false,
      };
    },
  };
});

function agentNamed(id: string, name: string): AgentChoice {
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
  } as unknown as AgentChoice;
}

const dispatch = agentNamed("agdef_dispatch", "Dispatch");
const billing = agentNamed("agdef_billing", "Billing assistant");

const QUESTION = "which invoices are past due?";

function billingFits(): AgentRecommendation {
  return { agent: billing, current: false, tools: ["list_invoices", "get_invoice"] };
}

function renderHome(onStart: DeskHomeProps["onStart"]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskHome
          agents={[dispatch, billing]}
          threads={[]}
          isLoading={false}
          isStarting={false}
          onStart={onStart}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function type(text: string) {
  const box = await screen.findByRole("textbox", { name: /Message/ });
  fireEvent.change(box, { target: { value: text } });

  return box;
}

function pressEnter(box: HTMLElement) {
  fireEvent.keyDown(box, { key: "Enter" });
}

beforeEach(() => {
  state.agent = dispatch;
  state.recommend = () => Promise.resolve(billingFits());
  state.asked = [];
});

describe("Desk home suggesting the agent with the tools for a question", () => {
  it("offers the agent with the tools at the top of the box once typing pauses", async () => {
    renderHome(vi.fn());

    await type(QUESTION);

    const offer = await screen.findByText(/has the tools for this/, {}, { timeout: 3000 });
    expect(offer).toHaveTextContent("Billing assistant has the tools for this");
    expect(screen.getByRole("button", { name: "Switch" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Keep Dispatch" })).toBeInTheDocument();
    expect(state.asked).toEqual([{ text: QUESTION, currentAgentId: dispatch.id }]);
  });

  it("asks the agent with the tools when the question is sent before the suggestion shows", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart).toHaveBeenCalledWith(billing.id, QUESTION, expect.anything());
    expect(screen.queryByText(/has the tools for this/)).toBeNull();
  });

  it("asks the agent with the tools when the suggestion was shown and nobody chose", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await screen.findByText(/has the tools for this/, {}, { timeout: 3000 });
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
  });

  it("keeps the person's agent once they say so, and stops offering the other", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    fireEvent.click(
      await screen.findByRole("button", { name: "Keep Dispatch" }, { timeout: 3000 }),
    );
    expect(screen.queryByText(/has the tools for this/)).toBeNull();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("switches to the suggested agent when the person asks it", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    fireEvent.click(await screen.findByRole("button", { name: "Switch" }, { timeout: 3000 }));
    expect(screen.getByRole("textbox", { name: "Message Billing assistant" })).toBeInTheDocument();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
  });

  it("asks the agent already chosen when no agent fits better", async () => {
    state.recommend = () => Promise.resolve(null);
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("does not hold a question up on a match that fails", async () => {
    state.recommend = () => Promise.reject(new Error("offline"));
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("sends to the agent already chosen once a slow match has been waited on long enough", async () => {
    state.recommend = () => new Promise(() => undefined);
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    expect(
      await screen.findByPlaceholderText("Finding the agent with the tools for this…"),
    ).toBeDisabled();
    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1), { timeout: 3000 });
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("asks the chosen agent its own starter question without matching it", async () => {
    state.agent = { ...billing, starters: [{ label: "Late", prompt: QUESTION }] };
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
  });

  it("does not match a question too short to name a task", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type("hi");
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
    expect(state.asked).toEqual([]);
  });
});
