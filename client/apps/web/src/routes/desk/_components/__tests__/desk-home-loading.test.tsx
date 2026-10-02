import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskHome, type DeskHomeProps } from "../desk-home";

const state = vi.hoisted(() => ({
  canDecide: true,
  permissionsLoading: false,
  briefing: { settled: true, value: null as unknown },
  summary: { isPending: false, total: 2 },
  agent: null as unknown,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" ? state.canDecide : true,
    isLoading: state.permissionsLoading,
  }),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    briefing: {
      today: () => ({
        queryKey: ["briefing-today", state.briefing.settled],
        queryFn: () =>
          state.briefing.settled ? state.briefing.value : new Promise(() => undefined),
      }),
    },
    assistant: {
      myAgents: () => ({ queryKey: ["my-agents"], queryFn: () => [] }),
    },
  },
}));

vi.mock("../decisions/use-pending-decisions", () => ({
  usePendingDecisionSummary: (enabled: boolean) =>
    enabled && state.summary.isPending
      ? { isPending: true, data: undefined }
      : {
          isPending: !enabled,
          data: enabled ? { total: state.summary.total, byAgent: [], oldestAt: null } : undefined,
        },
}));

vi.mock("@/hooks/use-attention", () => ({
  useAttentionSummary: () => ({ data: undefined }),
}));

vi.mock("@/components/assistant/use-askable-agent", () => ({
  useAskableAgent: () => ({
    agent: state.agent,
    choose: () => undefined,
    recency: { ids: [], lastUsedAt: new Map() },
    choices: { isLoading: false, isError: false, recent: [], items: [], refetch: () => undefined },
    noneAvailable: false,
  }),
}));

const agent = {
  id: "agent-1",
  name: "Dispatch",
  description: "Assigns drivers",
  starters: [{ label: "Late loads", prompt: "Which loads are late?" }],
} as unknown as DeskHomeProps["agents"][number];

function renderHome(props: Partial<DeskHomeProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskHome
          agents={[agent]}
          threads={[]}
          isLoading={false}
          isStarting={false}
          onStart={() => undefined}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * The greeting is known before anything loads; the line under it is the one
 * the figures decide, so that is the line held as a skeleton rather than said
 * and then taken back.
 */
async function expectHeadlineSkeleton() {
  const heading = await screen.findByRole("heading", { level: 1 });
  expect(heading).toHaveTextContent(/^Good (morning|afternoon|evening)/);
  const headline = document.querySelector('[data-slot="desk-headline"]');
  expect(headline?.querySelector('[data-slot="skeleton"]')).not.toBeNull();
  expect(headline).toHaveTextContent("");
  expect(screen.queryByText(/waiting on you/)).toBeNull();
  expect(screen.queryByText("How can I help today?")).toBeNull();
}

beforeEach(() => {
  state.canDecide = true;
  state.permissionsLoading = false;
  state.briefing = { settled: true, value: null };
  state.summary = { isPending: false, total: 2 };
  state.agent = agent;
});

describe("Desk home headline while its figures load", () => {
  it("holds the headline as a skeleton while the agents and conversations load", async () => {
    renderHome({ isLoading: true });

    await expectHeadlineSkeleton();
  });

  it("holds the headline as a skeleton while permissions are still arriving", async () => {
    state.permissionsLoading = true;
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("waits for the briefing, whose headline would replace the computed one", async () => {
    state.briefing = { settled: false, value: null };
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("waits for what is waiting on someone who decides", async () => {
    state.summary.isPending = true;
    renderHome();

    await expectHeadlineSkeleton();
  });

  it("does not wait on a decision summary someone may not see", async () => {
    state.canDecide = false;
    state.summary.isPending = true;
    renderHome();

    expect(await screen.findByText("How can I help today?")).toBeInTheDocument();
  });

  it("opens on the briefing's headline when the morning's briefing wrote one", async () => {
    state.briefing = { settled: true, value: { headline: "Three loads sit under a storm warning." } };
    renderHome();

    expect(await screen.findByText("Three loads sit under a storm warning.")).toBeInTheDocument();
  });

  it("opens on the headline its figures support once everything is in", async () => {
    renderHome();

    expect(await screen.findByText("2 decisions are waiting on you.")).toBeInTheDocument();
  });
});

describe("Desk home composer", () => {
  it("starts a conversation with the chosen agent and the question typed", async () => {
    const onStart = vi.fn();
    renderHome({ onStart });

    const box = await screen.findByRole("textbox", { name: "Message Dispatch" });
    fireEvent.change(box, { target: { value: "Who is free near Joliet?" } });
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onStart).toHaveBeenCalledWith("agent-1", "Who is free near Joliet?");
  });

  it("asks a starter question outright with its shortcut", async () => {
    const onStart = vi.fn();
    renderHome({ onStart });

    const box = await screen.findByRole("textbox", { name: "Message Dispatch" });
    fireEvent.keyDown(box, { key: "1", metaKey: true });

    expect(onStart).toHaveBeenCalledWith("agent-1", "Which loads are late?");
  });

  it("says no agents are available instead of offering a box nobody answers", async () => {
    renderHome({ agents: [] });

    expect(await screen.findByText("No agents are available.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).toBeNull();
  });
});
