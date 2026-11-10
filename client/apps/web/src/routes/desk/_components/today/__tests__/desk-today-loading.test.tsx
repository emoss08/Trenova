import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskToday, type DeskTodayProps } from "../desk-today";

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

vi.mock("@/lib/queries/briefing", () => ({
  briefing: {
    today: () => ({
      queryKey: ["briefing-today", state.briefing.settled],
      queryFn: () => (state.briefing.settled ? state.briefing.value : new Promise(() => undefined)),
    }),
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
    agentMatch: (text: string) => ({
      queryKey: ["agent-match", text],
      queryFn: () => ({ outcome: "NoMatch", agent: null, tools: [] }),
    }),
  },
}));

const pending = vi.hoisted(() => () => new Promise(() => undefined));

vi.mock("@/lib/queries/desk-attention", () => ({
  deskAttention: { list: () => ({ queryKey: ["desk-attention"], queryFn: pending }) },
}));
vi.mock("@/lib/queries/desk-activity", () => ({
  deskActivity: { since: () => ({ queryKey: ["desk-activity"], queryFn: pending }) },
}));
vi.mock("@/lib/queries/watchtower", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/watchtower")>();
  return {
    ...actual,
    watchtower: {
      ...actual.watchtower,
      counts: () => ({ queryKey: ["watchtower-counts"], queryFn: pending }),
      cleared: () => ({ queryKey: ["watchtower-cleared"], queryFn: pending }),
    },
  };
});
vi.mock("@/lib/queries/inbox", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/inbox")>();
  return {
    ...actual,
    inbox: {
      ...actual.inbox,
      briefing: () => ({ queryKey: ["inbox-briefing"], queryFn: pending }),
    },
  };
});

vi.mock("../../decisions/use-pending-decisions", () => ({
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

/** This morning's briefing as the schema returns it: sections, links and withheld are always lists. */
function briefingOf(fields: {
  headline: string;
  headlineLinks?: { text: string; section: string; path: string }[];
  withheldSections?: unknown[];
}) {
  return { sections: [], headlineLinks: [], withheldSections: [], ...fields };
}

const agent = {
  id: "agent-1",
  name: "Dispatch",
  description: "Assigns drivers",
  starters: [{ label: "Late loads", prompt: "Which loads are late?" }],
} as unknown as DeskTodayProps["agents"][number];

function renderHome(props: Partial<DeskTodayProps> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskToday
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
    state.briefing = {
      settled: true,
      value: briefingOf({ headline: "Three loads sit under a storm warning." }),
    };
    renderHome();

    expect(await screen.findByText("Three loads sit under a storm warning.")).toBeInTheDocument();
  });

  it("opens each phrase of the headline the briefing linked on the section it is about", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({
        headline: "142 loads are moving. Six may miss today's appointments, and 3 decisions wait.",
        headlineLinks: [
          { text: "Six may miss today's appointments", section: "Exceptions", path: "/desk/needs" },
          { text: "3 decisions wait", section: "Decisions", path: "/desk/decisions" },
        ],
      }),
    };
    renderHome();

    const risk = await screen.findByRole("link", { name: "Six may miss today's appointments" });
    expect(risk).toHaveAttribute("href", "/desk/needs");
    expect(screen.getByRole("link", { name: "3 decisions wait" })).toHaveAttribute(
      "href",
      "/desk/decisions",
    );
    expect(document.querySelector('[data-slot="desk-headline"]')).toHaveTextContent(
      "142 loads are moving. Six may miss today's appointments, and 3 decisions wait.",
    );
  });

  it("leaves a linked phrase the headline does not hold as plain text", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({
        headline: "Three loads sit under a storm warning.",
        headlineLinks: [{ text: "Four loads", section: "Exceptions", path: "/desk/needs" }],
      }),
    };
    renderHome();

    expect(await screen.findByText("Three loads sit under a storm warning.")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Four loads" })).toBeNull();
  });

  it("opens on the headline its figures support once everything is in", async () => {
    renderHome();

    expect(await screen.findByText("2 decisions are waiting on you.")).toBeInTheDocument();
  });
});

describe("Desk home briefing cut to the reader's access", () => {
  it("makes the count of decisions waiting a link to them", async () => {
    renderHome();

    const link = await screen.findByRole("link", { name: "2 decisions are waiting on you." });
    expect(link).toHaveAttribute("href", "/desk/decisions");
  });

  it("asks how it can help, with no link, when nothing waits", async () => {
    state.summary = { isPending: false, total: 0 };
    renderHome();

    expect(await screen.findByText("How can I help today?")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /waiting on you/ })).toBeNull();
  });

  it("says nothing about access when the reader sees the whole briefing", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({ headline: "Three loads sit under a storm warning." }),
    };
    renderHome();

    expect(await screen.findByText("Three loads sit under a storm warning.")).toBeInTheDocument();
    expect(document.querySelector('[data-slot="desk-briefing-withheld"]')).toBeNull();
  });

  it("names the sections left out and falls back to the computed headline", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({
        headline: "",
        withheldSections: [
          { key: "Billing", title: "Billing queue", partial: false },
          { key: "Cash", title: "Cash received yesterday", partial: false },
        ],
      }),
    };
    renderHome();

    expect(await screen.findByText("2 decisions are waiting on you.")).toBeInTheDocument();
    expect(
      screen.getByText(
        "This morning's briefing leaves out Billing queue and Cash received yesterday, whose records you can't open.",
      ),
    ).toBeInTheDocument();
  });

  it("says which sections it shows only in part", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({
        headline: "",
        withheldSections: [{ key: "Attention", title: "Needs attention", partial: true }],
      }),
    };
    renderHome();

    expect(
      await screen.findByText(
        "This morning's briefing shows only the part of Needs attention you can open.",
      ),
    ).toBeInTheDocument();
  });

  it("names both what it leaves out and what it shows in part", async () => {
    state.briefing = {
      settled: true,
      value: briefingOf({
        headline: "",
        withheldSections: [
          { key: "Exceptions", title: "Critical", partial: true },
          { key: "Cash", title: "Cash received yesterday", partial: false },
        ],
      }),
    };
    renderHome();

    expect(
      await screen.findByText(
        "This morning's briefing leaves out Cash received yesterday and shows only the part of Critical you can open.",
      ),
    ).toBeInTheDocument();
  });
});

describe("Desk home composer", () => {
  it("starts a conversation with the chosen agent and the question typed", async () => {
    const onStart = vi.fn();
    renderHome({ onStart });

    const box = await screen.findByRole("textbox", { name: "Message Dispatch" });
    fireEvent.change(box, { target: { value: "Who is free near Joliet?" } });
    fireEvent.keyDown(box, { key: "Enter" });

    await waitFor(() =>
      expect(onStart).toHaveBeenCalledWith("agent-1", "Who is free near Joliet?", {
        files: [],
        mentions: [],
        providerId: "",
      }),
    );
  });

  it("asks a starter question outright with its shortcut", async () => {
    const onStart = vi.fn();
    renderHome({ onStart });

    const box = await screen.findByRole("textbox", { name: "Message Dispatch" });
    fireEvent.keyDown(box, { key: "1", metaKey: true });

    expect(onStart).toHaveBeenCalledWith("agent-1", "Which loads are late?", {
      files: [],
      mentions: [],
      providerId: "",
    });
  });

  it("says no agents are available instead of offering a box nobody answers", async () => {
    renderHome({ agents: [] });

    expect(await screen.findByText("No agents are available.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).toBeNull();
  });
});
