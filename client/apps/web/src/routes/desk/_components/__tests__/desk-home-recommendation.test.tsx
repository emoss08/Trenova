import type { AgentChoice, AgentMatch } from "@/lib/graphql/agent-definition";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskHome, type DeskHomeProps } from "../desk-home";

const state = vi.hoisted(() => ({
  agent: null as unknown,
  recommend: (() => Promise.resolve({ outcome: "NoMatch", agent: null, tools: [] })) as (
    text: string,
    currentAgentId: string,
  ) => Promise<unknown>,
  asked: [] as { text: string; currentAgentId: string }[],
  contextKinds: [] as string[][],
  canAddTools: true,
  recorded: [] as {
    suggestedAgentId: string;
    currentAgentId: string | null;
    surface: string;
    outcome: string;
  }[],
  settled: [] as { id: string; outcome: string }[],
}));

vi.mock("@/hooks/use-permission", async () => {
  const { Operation } = await import("@trenova/shared/types/permission");

  return {
    usePermission: (_resource: unknown, operation: unknown) => ({
      allowed: operation === Operation.Update ? state.canAddTools : true,
      isLoading: false,
    }),
  };
});

vi.mock("@/lib/graphql/agent-suggestion", () => ({
  recordAgentSuggestion: (input: (typeof state.recorded)[number]) => {
    state.recorded.push(input);

    return Promise.resolve(`agsg_${state.recorded.length}`);
  },
  settleAgentSuggestion: (id: string, outcome: string) => {
    state.settled.push({ id, outcome });

    return Promise.resolve();
  },
}));

vi.mock("@/lib/queries/briefing", () => ({
  briefing: {
    today: () => ({ queryKey: ["briefing-today"], queryFn: () => null }),
  },
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    commands: (agentId: string) => ({
      queryKey: ["assistant-commands", agentId],
      queryFn: () => ({ commands: [] }),
    }),
    myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
    providers: () => ({ queryKey: ["providers"], queryFn: () => [] }),
    agentMatch: (text: string, currentAgentId: string | null, contextKinds: readonly string[]) => ({
      queryKey: ["agent-match", text, currentAgentId ?? "", contextKinds.join(",")],
      queryFn: ({ signal }: { signal: AbortSignal }) => {
        state.asked.push({ text, currentAgentId: currentAgentId ?? "" });
        state.contextKinds.push([...contextKinds]);
        // Like the real request, the answer is dropped when the query is cancelled.
        return new Promise((resolve, reject) => {
          signal.addEventListener("abort", () => reject(signal.reason), { once: true });
          state.recommend(text, currentAgentId ?? "").then(resolve, reject);
        });
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

function billingFits(): AgentMatch {
  return {
    outcome: "BetterAgent",
    agent: billing,
    tools: [
      { name: "list_invoices", summary: "List invoices by status or due date." },
      { name: "get_invoice", summary: "" },
    ],
  };
}

const NO_MATCH: AgentMatch = { outcome: "NoMatch", agent: null, tools: [] };

const NO_AGENT: AgentMatch = {
  outcome: "NoAgent",
  agent: null,
  tools: [{ name: "list_invoices", summary: "List invoices by status or due date." }],
};

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

const NOTE = '[data-slot="agent-recommendation"]';

function queryNote(): HTMLElement | null {
  return document.querySelector<HTMLElement>(NOTE);
}

async function findNote(): Promise<HTMLElement> {
  let note: HTMLElement | null = null;
  await waitFor(
    () => {
      note = queryNote();
      expect(note).not.toBeNull();
    },
    { timeout: 3000 },
  );

  return note as unknown as HTMLElement;
}

function pressEnter(box: HTMLElement) {
  fireEvent.keyDown(box, { key: "Enter" });
}

beforeEach(() => {
  state.agent = dispatch;
  state.recommend = () => Promise.resolve(billingFits());
  state.asked = [];
  state.contextKinds = [];
  state.canAddTools = true;
  state.recorded = [];
  state.settled = [];
  useDeskSettingsStore.getState().reset();
});

describe("Desk home suggesting the agent with the tools for a question", () => {
  it("offers the agent with the tools at the top of the box once typing pauses", async () => {
    renderHome(vi.fn());

    await type(QUESTION);

    const offer = await findNote();
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
    expect(queryNote()).toBeNull();
  });

  it("asks the agent with the tools when the suggestion was shown and nobody chose", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await findNote();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
  });

  it("takes the suggestion away with the question once it is sent", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await findNote();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(queryNote()).toBeNull();
  });

  it("keeps the person's agent once they say so, and stops offering the other", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    fireEvent.click(
      await screen.findByRole("button", { name: "Keep Dispatch" }, { timeout: 3000 }),
    );
    expect(queryNote()).toBeNull();
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

  it("offers again when a new question replaces the one switched for, and Enter follows it", async () => {
    const customerService = agentNamed("agdef_cs", "Customer service");
    const shipments = "Show me a list of shipments";
    state.recommend = (text, currentAgentId) =>
      Promise.resolve(
        text === shipments
          ? ({
              outcome: "BetterAgent",
              agent: customerService,
              tools: [{ name: "list_shipments", summary: "" }],
            } as AgentMatch)
          : currentAgentId === billing.id
            ? { ...billingFits(), outcome: "CurrentAgent" }
            : billingFits(),
      );
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    fireEvent.click(await screen.findByRole("button", { name: "Switch" }, { timeout: 3000 }));
    await new Promise((settle) => setTimeout(settle, 1000));
    expect(queryNote()).toBeNull();

    fireEvent.change(box, { target: { value: shipments } });

    expect(await findNote()).toHaveTextContent("Customer service has the tools for this");
    pressEnter(box);
    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(customerService.id);
  });

  it("turns down only the agent the cross was used on", async () => {
    const customerService = agentNamed("agdef_cs", "Customer service");
    const shipments = "Show me a list of shipments";
    state.recommend = (text) =>
      Promise.resolve(
        text === shipments
          ? ({
              outcome: "BetterAgent",
              agent: customerService,
              tools: [{ name: "list_shipments", summary: "" }],
            } as AgentMatch)
          : billingFits(),
      );
    renderHome(vi.fn());

    const box = await type(QUESTION);
    fireEvent.click(
      await screen.findByRole("button", { name: "Keep Dispatch" }, { timeout: 3000 }),
    );
    fireEvent.change(box, { target: { value: `${QUESTION} today` } });
    await new Promise((settle) => setTimeout(settle, 1000));
    expect(queryNote()).toBeNull();

    fireEvent.change(box, { target: { value: shipments } });

    expect(await findNote()).toHaveTextContent("Customer service has the tools for this");
  });

  it("suggests again for the next question once the box is cleared", async () => {
    renderHome(vi.fn());

    const box = await type(QUESTION);
    fireEvent.click(
      await screen.findByRole("button", { name: "Keep Dispatch" }, { timeout: 3000 }),
    );
    expect(queryNote()).toBeNull();
    fireEvent.change(box, { target: { value: "" } });
    fireEvent.change(box, { target: { value: QUESTION } });

    expect(await findNote()).toHaveTextContent("Billing assistant has the tools for this");
  });

  it("waits on a match already under way when the question is sent, though the box empties", async () => {
    state.recommend = () =>
      new Promise((settle) => {
        setTimeout(() => settle(billingFits()), 800);
      });
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await waitFor(() => expect(state.asked).toHaveLength(1), { timeout: 2000 });
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1), { timeout: 3000 });
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
    expect(state.asked).toHaveLength(1);
  });

  it("holds the suggestion's choices still while a question waits on its match", async () => {
    let finish: (value: unknown) => void = () => undefined;
    state.recommend = (_text, currentAgentId) =>
      state.asked.length > 1
        ? new Promise((settle) => {
            finish = settle;
          })
        : Promise.resolve(currentAgentId === dispatch.id ? billingFits() : NO_MATCH);
    renderHome(vi.fn());

    const box = await type(QUESTION);
    await findNote();
    fireEvent.change(box, { target: { value: `${QUESTION} now` } });
    pressEnter(box);

    expect(await screen.findByRole("button", { name: "Switch" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Keep Dispatch" })).toBeDisabled();
    finish(billingFits());
  });

  it("puts the cursor back in the box after Switch or the cross", async () => {
    renderHome(vi.fn());

    await type(QUESTION);
    fireEvent.click(await screen.findByRole("button", { name: "Switch" }, { timeout: 3000 }));

    await waitFor(() =>
      expect(screen.getByRole("textbox", { name: "Message Billing assistant" })).toHaveFocus(),
    );
  });

  it("announces the suggestion in a region that is always there", async () => {
    renderHome(vi.fn());

    const status = screen
      .getAllByRole("status")
      .find((region) => region.getAttribute("aria-live") === "polite" && region.textContent === "");
    expect(status).toBeDefined();
    await type(QUESTION);

    await waitFor(
      () => expect(status).toHaveTextContent("Billing assistant has the tools for this"),
      { timeout: 3000 },
    );
  });

  it("asks the agent already chosen when no agent fits better", async () => {
    state.recommend = () => Promise.resolve(NO_MATCH);
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

  it("says why the agent fits: the tools that matched, none of which the agent chosen holds", async () => {
    renderHome(vi.fn());

    await type(QUESTION);
    await findNote();
    fireEvent.click(screen.getByRole("button", { name: "Why Billing assistant" }));

    const why = await waitFor(() => {
      const found = document.querySelector<HTMLElement>('[data-slot="agent-recommendation-why"]');
      expect(found).not.toBeNull();
      return found as HTMLElement;
    });
    expect(why).toHaveTextContent("Why Billing assistant");
    expect(why).toHaveTextContent("List invoices by status or due date.");
    expect(why).toHaveTextContent("Dispatch holds none of these.");
    expect(why.querySelectorAll("li")).toHaveLength(2);
  });

  it("tells someone who can add tools that none of their agents has them, and where to add them", async () => {
    state.recommend = () => Promise.resolve(NO_AGENT);
    renderHome(vi.fn());

    await type(QUESTION);

    const note = await waitFor(
      () => {
        const found = document.querySelector<HTMLElement>(
          '[data-slot="agent-recommendation-none"]',
        );
        expect(found).not.toBeNull();
        return found as HTMLElement;
      },
      { timeout: 3000 },
    );
    expect(note).toHaveTextContent("None of your agents has the tools for this.");
    expect(screen.getByRole("link", { name: "Add tools" })).toHaveAttribute(
      "href",
      "/admin/agent-control?tab=agents",
    );
  });

  it("tells anyone else that none of their agents has the tools and who can add them", async () => {
    state.recommend = () => Promise.resolve(NO_AGENT);
    state.canAddTools = false;
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);

    const note = await waitFor(
      () => {
        const found = document.querySelector<HTMLElement>(
          '[data-slot="agent-recommendation-none"]',
        );
        expect(found).not.toBeNull();
        return found as HTMLElement;
      },
      { timeout: 3000 },
    );
    expect(note).toHaveTextContent(
      "None of your agents has the tools for this. An administrator can add them.",
    );
    expect(screen.queryByRole("link", { name: "Add tools" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(document.querySelector('[data-slot="agent-recommendation-none"]')).toBeNull();
    pressEnter(box);
    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("only names the better agent when the person asked to be told, and sends to the one chosen", async () => {
    useDeskSettingsStore.getState().set("suggestAgent", "suggest");
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await findNote();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "PassedOver" }]));
  });

  it("moves the question to the agent with the tools without a note when auto switch is on", async () => {
    useDeskSettingsStore.getState().set("autoSwitchAgent", "on");
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await waitFor(() => expect(state.asked).toHaveLength(1), { timeout: 2000 });
    await new Promise((settle) => setTimeout(settle, 300));
    expect(queryNote()).toBeNull();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
    expect(state.recorded).toEqual([
      {
        suggestedAgentId: billing.id,
        currentAgentId: dispatch.id,
        surface: "Home",
        outcome: "AutoSwitched",
      },
    ]);
  });

  it("matches nothing when the person turned suggestions off", async () => {
    useDeskSettingsStore.getState().set("suggestAgent", "off");
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await new Promise((settle) => setTimeout(settle, 1000));
    expect(state.asked).toEqual([]);
    expect(queryNote()).toBeNull();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
  });

  it("keeps the agent the person starts with, unless auto switch moves the question", async () => {
    useDeskSettingsStore.getState().set("agent", dispatch.id);
    const onStart = vi.fn();
    const { unmount } = renderHome(onStart);

    let box = await type(QUESTION);
    await findNote();
    pressEnter(box);
    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(dispatch.id);
    unmount();

    useDeskSettingsStore.getState().set("autoSwitchAgent", "on");
    renderHome(onStart);
    box = await type(QUESTION);
    pressEnter(box);
    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(2));
    expect(onStart.mock.calls[1]?.[0]).toBe(billing.id);
  });

  it("records a suggestion once while it stays on screen, and what the person did with it", async () => {
    renderHome(vi.fn());

    const box = await type(QUESTION);
    await findNote();
    fireEvent.change(box, { target: { value: `${QUESTION} please` } });
    await new Promise((settle) => setTimeout(settle, 900));
    fireEvent.click(screen.getByRole("button", { name: "Switch" }));

    expect(state.recorded).toEqual([
      {
        suggestedAgentId: billing.id,
        currentAgentId: dispatch.id,
        surface: "Home",
        outcome: "Shown",
      },
    ]);
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "Switched" }]));
  });

  it("records the agent kept with the cross", async () => {
    renderHome(vi.fn());

    await type(QUESTION);
    fireEvent.click(
      await screen.findByRole("button", { name: "Keep Dispatch" }, { timeout: 3000 }),
    );

    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "Kept" }]));
  });

  it("records a question sent to the suggested agent as sent through it", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    await findNote();
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(onStart.mock.calls[0]?.[0]).toBe(billing.id);
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "SentThrough" }]));
  });

  it("records a question switch on send moved before its note showed as shown and sent through", async () => {
    const onStart = vi.fn();
    renderHome(onStart);

    const box = await type(QUESTION);
    pressEnter(box);

    await waitFor(() => expect(onStart).toHaveBeenCalledTimes(1));
    expect(state.recorded).toEqual([
      expect.objectContaining({ suggestedAgentId: billing.id, outcome: "Shown" }),
    ]);
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "SentThrough" }]));
  });

  it("asks for a match about nothing in particular when no record is named", async () => {
    renderHome(vi.fn());

    await type(QUESTION);

    await waitFor(() => expect(state.contextKinds).toEqual([[]]), { timeout: 2000 });
  });
});
