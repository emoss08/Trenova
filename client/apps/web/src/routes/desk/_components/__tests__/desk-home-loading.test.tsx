import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskHome, type DeskHomeProps } from "../desk-home";

const state = vi.hoisted(() => ({
  canDecide: true,
  permissionsLoading: false,
  briefing: { settled: true, value: null as unknown },
  summary: { isPending: false, total: 2 },
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

vi.mock("@/components/assistant/use-active-turns", () => ({
  useLiveThreadIds: () => new Set<string>(),
}));

vi.mock("@/components/assistant/use-askable-agent", () => ({
  useAskableAgent: () => ({
    agent: null,
    choose: () => undefined,
    recency: { ids: [], lastUsedAt: {} },
    choices: { isLoading: true, isError: false, recent: [], items: [], refetch: () => undefined },
    noneAvailable: false,
  }),
}));

vi.mock("../desk-agent-directory", () => ({ DeskAgentDirectory: () => null }));
vi.mock("../desk-decisions-callout", () => ({ DeskDecisionsCallout: () => null }));
vi.mock("../desk-recent-conversations", () => ({ DeskRecentConversations: () => null }));
vi.mock("../briefing-panel", () => ({ BriefingPanel: () => null }));

const agent = { id: "agent-1", name: "Dispatch" } as DeskHomeProps["agents"][number];

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

async function expectSettingOut() {
  const status = await screen.findByRole("status");
  expect(status).toHaveTextContent("Opening the desk");
  expect(screen.queryByText(/waiting on you/)).toBeNull();
}

beforeEach(() => {
  state.canDecide = true;
  state.permissionsLoading = false;
  state.briefing = { settled: true, value: null };
  state.summary = { isPending: false, total: 2 };
});

describe("Desk home while it is set out", () => {
  it("shows the desk visitor while the agents and conversations load", async () => {
    renderHome({ isLoading: true });

    await expectSettingOut();
  });

  it("shows the desk visitor while permissions are still arriving", async () => {
    state.permissionsLoading = true;
    renderHome();

    await expectSettingOut();
  });

  it("waits for the briefing, whose headline would replace the computed one", async () => {
    state.briefing = { settled: false, value: null };
    renderHome();

    await expectSettingOut();
  });

  it("waits for what is waiting on someone who decides", async () => {
    state.summary.isPending = true;
    renderHome();

    await expectSettingOut();
  });

  it("does not wait on a decision summary someone may not see", async () => {
    state.canDecide = false;
    state.summary.isPending = true;
    renderHome();

    expect(await screen.findByText("Nothing is waiting on you.")).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("opens on the headline its figures support once everything is in", async () => {
    renderHome();

    expect(await screen.findByText("2 decisions are waiting on you.")).toBeInTheDocument();
    expect(screen.queryByText("Opening the desk")).toBeNull();
  });
});
