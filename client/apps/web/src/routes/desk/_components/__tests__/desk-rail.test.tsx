import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskRail, type DeskRailProps } from "../desk-rail";

const DAY = 24 * 60 * 60;

const state = vi.hoisted(() => ({
  canDecide: true,
  canWatch: true,
  decisions: 0,
  live: new Set<string>(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" ? state.canDecide : state.canWatch,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/use-attention", () => ({
  useAttentionSummary: () => ({ data: { agentDecisions: state.decisions } }),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    watchtower: {
      counts: () => ({
        queryKey: ["watchtower-counts"],
        queryFn: () => ({ unseen: 3, unseenCritical: 0 }),
      }),
    },
    user: {
      profilePicture: () => ({ queryKey: ["profile-picture"], queryFn: () => null }),
    },
  },
}));

vi.mock("@/components/assistant/use-active-turns", () => ({
  useLiveThreadIds: () => state.live,
}));

vi.mock("@/components/assistant/agent-picker", () => ({
  AgentPicker: ({ trigger }: { trigger: React.ReactElement }) => trigger,
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({
      user: {
        id: "usr_1",
        name: "Avery Lane",
        username: "avery",
        emailAddress: "avery@example.com",
        timezone: "UTC",
      },
    }),
}));

const agents = [
  { id: "agtd_1", name: "Billing desk", starters: [] },
  { id: "agtd_2", name: "Dispatch desk", starters: [] },
] as unknown as AgentChoice[];

function thread(overrides: Partial<AssistantThread> & { id: string }): AssistantThread {
  const now = Math.floor(Date.now() / 1000);
  return {
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agtd_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: "Blocked invoices",
    status: "Active",
    lastMessageAt: now - 60,
    version: 0,
    createdAt: now - 120,
    updatedAt: now - 60,
    ...overrides,
  };
}

function renderRail(props: Partial<DeskRailProps> = {}, path = "/desk") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = {
    onStart: vi.fn(),
    onDelete: vi.fn(),
    onTogglePin: vi.fn(),
    onRetry: vi.fn(),
    onCollapse: vi.fn(),
    onNavigate: vi.fn(),
  };

  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <DeskRail
          threads={[]}
          agents={agents}
          activeThreadId={null}
          isLoading={false}
          listUnavailable={false}
          isStarting={false}
          collapse={{ label: "Hide the rail", shortcut: "⌘B" }}
          {...handlers}
          {...props}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { ...view, handlers };
}

beforeEach(() => {
  state.canDecide = true;
  state.canWatch = true;
  state.decisions = 0;
  state.live = new Set();
});

/**
 * The rail is the Desk's table of contents: the way in, the places, and
 * every conversation shelved by when it was last touched, what the person
 * pinned above the calendar. Each row names the agent it was with, falls
 * back to a name when the conversation has none, and keeps its pin and
 * delete within reach of a keyboard even though a pointer only sees them
 * on hover.
 */
describe("DeskRail", () => {
  it("shelves conversations by recency with pinned ones first, titles alone on each row", () => {
    const now = Math.floor(Date.now() / 1000);
    renderRail({
      threads: [
        thread({ id: "a", title: "Fuel surcharge", lastMessageAt: now - 12 * DAY }),
        thread({ id: "b", title: "", agentDefinitionId: "agtd_2" }),
        thread({ id: "c", title: "Late loads", pinned: true, lastMessageAt: now - 3 * DAY }),
      ],
    });

    const shelves = screen.getAllByRole("region").map((shelf) => shelf.getAttribute("aria-label"));
    expect(shelves).toEqual(["Pinned", "Today", "Previous 30 days"]);

    const today = screen.getByRole("region", { name: "Today" });
    const row = within(today).getByRole("link", { name: /Untitled conversation/ });
    expect(row).toHaveAttribute("href", "/desk/t/b");
    expect(row).toHaveTextContent(/^Untitled conversation$/);
    expect(row).toHaveAttribute("title", "With Dispatch desk");
  });

  it("marks the open conversation and says when a reply is being written", () => {
    state.live = new Set(["a"]);
    renderRail({ threads: [thread({ id: "a" })], activeThreadId: "a" }, "/desk/t/a");

    const row = screen.getByRole("link", { name: /Blocked invoices/ });
    expect(row).toHaveAttribute("aria-current", "page");
    expect(row).toHaveTextContent("Replying");
  });

  it("narrows the list to the search, by title or agent, and says when nothing matches", () => {
    renderRail({
      threads: [
        thread({ id: "a", title: "Fuel surcharge" }),
        thread({ id: "b", title: "Driver roster", agentDefinitionId: "agtd_2" }),
      ],
    });

    const search = screen.getByRole("textbox", { name: "Search conversations" });
    fireEvent.change(search, { target: { value: "dispatch" } });
    expect(screen.queryByRole("link", { name: /Fuel surcharge/ })).toBeNull();
    expect(screen.getByRole("link", { name: /Driver roster/ })).toBeInTheDocument();

    fireEvent.change(search, { target: { value: "nothing like this" } });
    expect(screen.getByText("No conversations match that search.")).toBeInTheDocument();
  });

  it("hands the row's pin and delete to the room", async () => {
    const item = thread({ id: "a" });
    const { handlers } = renderRail({ threads: [item] });

    fireEvent.click(screen.getByRole("button", { name: "Conversation actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Pin conversation" }));
    expect(handlers.onTogglePin).toHaveBeenCalledWith(item);

    fireEvent.click(screen.getByRole("button", { name: "Conversation actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Delete conversation" }));
    expect(handlers.onDelete).toHaveBeenCalledWith(item);
  });

  it("folds from its own header and names the keystroke", () => {
    const { handlers } = renderRail();

    const fold = screen.getByRole("button", { name: "Hide the rail" });
    expect(fold).toHaveAttribute("aria-keyshortcuts", "Meta+B Control+B");
    fireEvent.click(fold);
    expect(handlers.onCollapse).toHaveBeenCalledTimes(1);
  });

  it("shows the places a person may go, with what is waiting at each", async () => {
    state.decisions = 4;
    renderRail();

    const nav = screen.getByRole("navigation", { name: "Desk" });
    expect(within(nav).getByRole("link", { name: /Today/ })).toHaveAttribute("href", "/desk");
    expect(within(nav).getByRole("link", { name: /Decisions/ })).toHaveTextContent("4");
    const watchtower = within(nav).getByRole("link", { name: /Watchtower/ });
    expect(await within(watchtower).findByText("3")).toBeInTheDocument();
  });

  it("leaves out the places a person may not open", () => {
    state.canDecide = false;
    state.canWatch = false;
    renderRail();

    const nav = screen.getByRole("navigation", { name: "Desk" });
    expect(within(nav).queryByRole("link", { name: /Decisions/ })).toBeNull();
    expect(within(nav).queryByRole("link", { name: /Watchtower/ })).toBeNull();
  });

  it("holds its place while the list loads, and offers to try again when it could not", () => {
    const { unmount } = renderRail({ isLoading: true });
    expect(document.querySelector('[aria-busy] [data-slot="skeleton"]')).not.toBeNull();
    unmount();

    const { handlers } = renderRail({ listUnavailable: true });
    expect(screen.getByText("The conversations could not be loaded.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(handlers.onRetry).toHaveBeenCalledTimes(1);
  });

  it("says who is signed in and offers the way back to Trenova", () => {
    renderRail();

    expect(screen.getByText("Avery Lane")).toBeInTheDocument();
    expect(screen.getByText("avery@example.com")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to Trenova" })).toHaveAttribute("href", "/");
  });
});
