import type { AgentChoice, AgentMatch } from "@/lib/graphql/agent-definition";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useConversationSuggestion } from "../use-conversation-suggestion";

const state = vi.hoisted(() => ({
  answer: null as unknown,
  asked: [] as { text: string; currentAgentId: string; contextKinds: string }[],
  recorded: [] as { suggestedAgentId: string; outcome: string; surface: string }[],
  settled: [] as { id: string; outcome: string }[],
}));

vi.mock("@/lib/queries/assistant", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/queries/assistant")>()),
  assistant: {
    agentMatch: (text: string, currentAgentId: string | null, contextKinds: readonly string[]) => ({
      queryKey: ["agent-match", text, currentAgentId ?? "", contextKinds.join(",")],
      queryFn: () => {
        state.asked.push({
          text,
          currentAgentId: currentAgentId ?? "",
          contextKinds: contextKinds.join(","),
        });
        return Promise.resolve(state.answer);
      },
    }),
  },
}));

vi.mock("@/lib/graphql/agent-suggestion", () => ({
  recordAgentSuggestion: (input: (typeof state.recorded)[number]) => {
    state.recorded.push({
      suggestedAgentId: input.suggestedAgentId,
      outcome: input.outcome,
      surface: input.surface,
    });
    return Promise.resolve(`agsg_${state.recorded.length}`);
  },
  settleAgentSuggestion: (id: string, outcome: string) => {
    state.settled.push({ id, outcome });
    return Promise.resolve();
  },
}));

function agentNamed(id: string, name: string): AgentChoice {
  return { id, name, starters: [] } as unknown as AgentChoice;
}

const dispatch = agentNamed("agdef_dispatch", "Dispatch");
const billing = agentNamed("agdef_billing", "Billing");
const REPLY = "which invoices are past due for Acme?";

function billingFits(): AgentMatch {
  return {
    outcome: "BetterAgent",
    agent: billing,
    tools: [{ name: "list_invoices", summary: "List invoices by status." }],
  };
}

type Props = Parameters<typeof useConversationSuggestion>[0];

function renderSuggestion(overrides: Partial<Props> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onHandOff = vi.fn(() => Promise.resolve());
  const clearDraft = vi.fn();
  const props: Props = {
    agent: dispatch,
    draft: REPLY,
    mentions: [{ type: "customer", id: "cus_1", label: "Acme" }],
    page: null,
    attachmentCount: 0,
    busy: false,
    onHandOff,
    clearDraft,
    ...overrides,
  };
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  const hook = renderHook((current: Props) => useConversationSuggestion(current), {
    initialProps: props,
    wrapper,
  });

  return { ...hook, onHandOff, clearDraft, props };
}

beforeEach(() => {
  state.answer = billingFits();
  state.asked = [];
  state.recorded = [];
  state.settled = [];
  useDeskSettingsStore.getState().reset();
});

describe("useConversationSuggestion", () => {
  it("offers the agent with the tools for the reply, matched on the records it names", async () => {
    const { result } = renderSuggestion();

    await waitFor(() => expect(result.current.offered?.agent.id).toBe(billing.id), {
      timeout: 2000,
    });
    expect(state.asked[0]).toEqual({
      text: REPLY,
      currentAgentId: dispatch.id,
      contextKinds: "customer",
    });
    expect(state.recorded).toEqual([
      { suggestedAgentId: billing.id, outcome: "Shown", surface: "Conversation" },
    ]);
  });

  it("hands the conversation off with the reply and the records it names", async () => {
    const { result, onHandOff, clearDraft, props } = renderSuggestion();
    await waitFor(() => expect(result.current.offered).not.toBeNull(), { timeout: 2000 });

    await act(() => result.current.handOff());

    expect(onHandOff).toHaveBeenCalledWith(billing.id, REPLY, props.mentions);
    expect(clearDraft).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "Switched" }]));
  });

  it("keeps the reply in the box when the hand-off fails", async () => {
    const onHandOff = vi.fn(() => Promise.reject(new Error("refused")));
    const { result, clearDraft } = renderSuggestion({ onHandOff });
    await waitFor(() => expect(result.current.offered).not.toBeNull(), { timeout: 2000 });

    await act(() => result.current.handOff());

    expect(clearDraft).not.toHaveBeenCalled();
    expect(result.current.handingOff).toBe(false);
    await new Promise((settle) => setTimeout(settle, 0));
    expect(state.settled).toEqual([]);

    act(() => result.current.keep());
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "Kept" }]));
  });

  it("records no automatic switch when the hand-off it tried fails", async () => {
    useDeskSettingsStore.getState().set("autoSwitchAgent", "on");
    const onHandOff = vi.fn(() => Promise.reject(new Error("refused")));
    const { result } = renderSuggestion({ onHandOff });
    await waitFor(() => expect(state.asked).toHaveLength(1), { timeout: 2000 });

    let handed = true;
    await act(async () => {
      handed = await result.current.route(REPLY, []);
    });

    expect(handed).toBe(false);
    expect(onHandOff).toHaveBeenCalledTimes(1);
    expect(state.recorded).toEqual([]);
  });

  it("matches nothing while the agent works, files are attached or the conversation cannot be handed off", async () => {
    renderSuggestion({ busy: true });
    renderSuggestion({ attachmentCount: 1 });
    renderSuggestion({ onHandOff: undefined });

    await new Promise((settle) => setTimeout(settle, 900));
    expect(state.asked).toEqual([]);
  });

  it("hands a reply off without asking when auto switch is on", async () => {
    useDeskSettingsStore.getState().set("autoSwitchAgent", "on");
    const { result, onHandOff } = renderSuggestion();
    await waitFor(() => expect(state.asked).toHaveLength(1), { timeout: 2000 });
    expect(result.current.offered).toBeNull();

    let handed = false;
    await act(async () => {
      handed = await result.current.route(REPLY, []);
    });

    expect(handed).toBe(true);
    expect(onHandOff).toHaveBeenCalledWith(billing.id, REPLY, []);
    expect(state.recorded).toEqual([
      { suggestedAgentId: billing.id, outcome: "AutoSwitched", surface: "Conversation" },
    ]);
  });

  it("leaves the reply to this conversation unless auto switch is on", async () => {
    const { result, onHandOff } = renderSuggestion();
    await waitFor(() => expect(result.current.offered).not.toBeNull(), { timeout: 2000 });

    let handed = true;
    await act(async () => {
      handed = await result.current.route(REPLY, []);
    });
    result.current.sentHere();

    expect(handed).toBe(false);
    expect(onHandOff).not.toHaveBeenCalled();
    await waitFor(() => expect(state.settled).toEqual([{ id: "agsg_1", outcome: "PassedOver" }]));
  });

  it("leaves a reply the conversation's agent fits to it, even with auto switch on", async () => {
    useDeskSettingsStore.getState().set("autoSwitchAgent", "on");
    state.answer = { outcome: "CurrentAgent", agent: dispatch, tools: [] } satisfies AgentMatch;
    const { result, onHandOff } = renderSuggestion();

    let handed = true;
    await act(async () => {
      handed = await result.current.route(REPLY, []);
    });

    expect(handed).toBe(false);
    expect(onHandOff).not.toHaveBeenCalled();
  });

  it("says when none of the person's agents holds the tools the reply needs", async () => {
    state.answer = { outcome: "NoAgent", agent: null, tools: [] } satisfies AgentMatch;
    const { result } = renderSuggestion();

    await waitFor(() => expect(result.current.nobody).toBe(true), { timeout: 2000 });
    expect(result.current.offered).toBeNull();

    act(() => result.current.dismissNobody());
    expect(result.current.nobody).toBe(false);
  });
});
