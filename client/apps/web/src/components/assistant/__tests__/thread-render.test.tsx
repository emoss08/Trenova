import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, waitFor } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useAssistantStore } from "@/stores/assistant-store";
import {
  assistantMessageSchema,
  assistantPlanListSchema,
  assistantProposalListSchema,
  assistantThreadSchema,
  type AssistantMessage,
} from "@/types/assistant";
import { MessageThread, openingStagger } from "../message-thread";
import { advanceTurn, initialTurnState, type TurnState } from "../turn-stream";

type Row = { key: string; render: () => ReactNode };

const api = vi.hoisted(() => ({
  listProposals: vi.fn(),
  listPlans: vi.fn(),
  listProviders: vi.fn(),
}));
const live = vi.hoisted(() => ({ turn: null as TurnState | null }));
const drawn = vi.hoisted(() => ({ rows: [] as Row[][] }));

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      listProposals: api.listProposals,
      listPlans: api.listPlans,
      listProviders: api.listProviders,
    },
  },
}));

const history = vi.hoisted(() => ({ messages: [] as unknown[] }));

vi.mock("../use-thread-history", () => ({
  useThreadHistory: () => ({
    messages: history.messages,
    isLoading: false,
    hasOlder: false,
    isLoadingOlder: false,
    loadOlder: vi.fn(),
    total: history.messages.length,
    limit: 0,
    length: { state: "open", remaining: Number.POSITIVE_INFINITY },
  }),
}));

const send = vi.fn();
const rejoin = vi.fn();
const dismiss = vi.fn();

vi.mock("../use-assistant-turn", () => ({
  useAssistantTurn: () => ({
    turn: live.turn,
    isActive: live.turn !== null,
    send,
    rejoin,
    stop: vi.fn(),
    dismiss,
    retry: undefined,
  }),
}));

vi.mock("@/components/ai-feedback/feedback-control", () => ({ FeedbackControl: () => null }));
vi.mock("../use-active-turns", () => ({ useLiveThreadIds: () => new Set<string>() }));
vi.mock("../use-page-context", () => ({ usePageContext: () => () => null }));
vi.mock("../use-composer-context", () => ({
  useComposerContext: () => ({
    attachments: [],
    attachFiles: vi.fn(),
    removeAttachment: vi.fn(),
    mentions: [],
    setMentions: vi.fn(),
    searchMentions: vi.fn(async () => []),
    clear: vi.fn(),
  }),
}));

// The window lays out nothing without a size; the rows it was handed are
// what this test reads.
vi.mock("../virtual-thread", () => ({
  VirtualThread: ({ rows }: { rows: Row[] }) => {
    drawn.rows.push(rows);
    return null;
  },
}));

const thread = assistantThreadSchema.parse({
  id: "athr_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: "usr_me",
  agentDefinitionId: "agdef_1",
  title: "Dispatch",
  status: "Active",
  canContinue: true,
  createdAt: 10,
  updatedAt: 10,
});

const agent = {
  id: "agdef_1",
  name: "Dispatch assistant",
  description: "",
  starters: [],
} as unknown as AgentChoice;

function message(overrides: Record<string, unknown>): AssistantMessage {
  return assistantMessageSchema.parse({
    threadId: "athr_1",
    createdAt: 1_790_000_000,
    ...overrides,
  });
}

beforeEach(() => {
  history.messages = [
    message({ id: "amsg_1", sequence: 1, role: "User", content: "Where is S1?" }),
    message({ id: "amsg_2", sequence: 2, role: "Assistant", content: "In Dallas." }),
  ];
  api.listProposals.mockResolvedValue(assistantProposalListSchema.parse({ results: [] }));
  api.listPlans.mockResolvedValue(assistantPlanListSchema.parse({ results: [] }));
  api.listProviders.mockResolvedValue([]);
  live.turn = null;
  drawn.rows = [];
  useAssistantStore.setState({ deferredDecisions: [], decisionFocus: {}, drafts: {} });
  useAuthStore.setState({ user: { id: "usr_me", timezone: "UTC" } as never });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function tree(client: QueryClient) {
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <MessageThread thread={thread} agent={agent} expanded />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function rowsByKey(rows: Row[]): Map<string, Row> {
  return new Map(rows.map((row) => [row.key, row]));
}

/**
 * Every token of a reply re-rendered the whole visible thread: the rows were
 * one memo keyed on the turn, so each delta rebuilt every saved message's row
 * and the window drew all of them again. A saved row is the same row however
 * far the reply in progress has got.
 */
describe("the thread while a reply streams", () => {
  it("keeps every saved row as it was and moves only the reply in progress", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const startedAt = 1_790_000_100_000;
    let state = advanceTurn(initialTurnState("And S2?", null, { startedAt }), {
      event: "accepted",
      data: { content: "And S2?", scopeStage: "", scopeCategory: "" },
    });
    state = advanceTurn(state, { event: "delta", data: { text: "S2 is" } });
    live.turn = state;

    const view = render(tree(client));
    await waitFor(() => expect(api.listProposals).toHaveBeenCalled());
    await waitFor(() => expect(api.listPlans).toHaveBeenCalled());
    view.rerender(tree(client));
    const before = rowsByKey(drawn.rows.at(-1) ?? []);

    live.turn = advanceTurn(state, { event: "delta", data: { text: " in Austin." } });
    view.rerender(tree(client));
    const after = rowsByKey(drawn.rows.at(-1) ?? []);

    const saved = [...before.keys()].filter((key) => key !== "turn-in-progress");
    expect(saved.length).toBeGreaterThan(0);
    for (const key of saved) {
      expect(after.get(key)?.render, key).toBe(before.get(key)?.render);
    }
    expect(after.get("turn-in-progress")?.render).not.toBe(before.get("turn-in-progress")?.render);
  });
});

/**
 * The rows a thread opens on rise one after another from the top of the
 * window down. The list is anchored to its end, so the last few rows are
 * the ones in view; the rest are above the window and rise at once.
 */
describe("openingStagger", () => {
  it("staggers the last rows top down and leaves the rest without a wait", () => {
    const delays = openingStagger(["m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8"]);

    expect(delays.get("m1")).toBe(0);
    expect(delays.get("m2")).toBe(0);
    expect(delays.get("m3")).toBe(0);
    expect(delays.get("m4")).toBe(30);
    expect(delays.get("m8")).toBe(150);
  });

  it("starts from the first row of a short thread", () => {
    const delays = openingStagger(["m1", "m2"]);

    expect(delays.get("m1")).toBe(0);
    expect(delays.get("m2")).toBe(30);
  });
});
