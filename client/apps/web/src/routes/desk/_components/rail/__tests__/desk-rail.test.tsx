import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { stubLayout } from "@/test/layout";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  DeskRailShelf,
  DeskShelfCount,
  DeskShelfPaging,
} from "@/components/desk-chat/rail/desk-rail-rows";
import {
  DESK_SHELF_KEYS,
  deskShelfEdges,
  groupDeskThreadsByShelf,
  type DeskShelfKey,
} from "@/components/desk-chat/rail/desk-threads";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useDeskRailSlotStore } from "@/stores/desk-rail-slot-store";
import { DeskRail, type DeskRailProps, railViewOf } from "../desk-rail";

const DAY = 24 * 60 * 60;

const state = vi.hoisted(() => ({ live: new Set<string>() }));
/** How many requests are scheduled in each conversation, as the server lists them. */
const scheduled = vi.hoisted(() => ({ byThread: new Map<string, number>() }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      get: vi.fn(async (url: string) => {
        const threadId = /^\/assistant\/threads\/([^/]+)\/schedules\//.exec(url)?.[1];
        if (threadId === undefined) {
          throw new Error(`unexpected read of ${url}`);
        }
        const items = Array.from({ length: scheduled.byThread.get(threadId) ?? 0 }, (_, index) => ({
          id: `csch_${index}`,
          threadId,
          userId: "usr_1",
          prompt: `Request ${index + 1}`,
          cadence: "stored",
          cronExpression: "0 8 * * 1",
          enabled: true,
          createdAt: 0,
        }));
        return { items, total: items.length };
      }),
    },
  };
});

vi.mock("@/components/assistant/use-active-turns", () => ({
  useLiveThreadIds: () => state.live,
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (s: { user: unknown }) => unknown) =>
    selector({ user: { id: "usr_1", name: "Avery Lane", username: "avery", timezone: "UTC" } }),
}));

const agents = new Map(
  [
    { id: "agtd_1", name: "Billing desk", starters: [] },
    { id: "agtd_2", name: "Dispatch desk", starters: [] },
  ].map((agent) => [agent.id, agent as unknown as AgentChoice]),
);

// A conversation is made before its first message, so it is never created after its last one.
function thread(overrides: Partial<AssistantThread> & { id: string }): AssistantThread {
  const now = Math.floor(Date.now() / 1000);
  const lastMessageAt = overrides.lastMessageAt ?? now - 60;
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
    lastMessageAt,
    version: 0,
    createdAt: Math.min(now - 120, lastMessageAt),
    updatedAt: now - 60,
    ...overrides,
  };
}

type RailOptions = Partial<
  Omit<DeskRailProps, "shelves" | "today" | "noConversations" | "onLoadMore">
> & {
  threads?: AssistantThread[];
  /** How each shelf is being read; every shelf read to its end by default. */
  paging?: Partial<Record<DeskShelfKey, Partial<DeskShelfPaging>>>;
  /** What the server counted on each shelf; what was read of it by default. */
  counts?: Partial<Record<DeskShelfKey, DeskShelfCount>>;
};

/** When the rail is drawn: each test's conversations are timed against the moment it runs. */
const HARNESS_NOW = Math.floor(Date.now() / 1000);

const READ_TO_END: DeskShelfPaging = { loading: false, hasMore: false, failed: false };

/**
 * The rail as the Desk draws it: the conversations put on their shelves the
 * way the shelves' own reads would hand them back, folded as the person left
 * them, so a heading pressed in a test folds its shelf as it does on the Desk.
 */
function RailHarness({
  threads = [],
  paging = {},
  counts = {},
  onLoadMore,
  ...props
}: RailOptions & {
  onLoadMore: DeskRailProps["onLoadMore"];
  handlers: Pick<DeskRailProps, "onSearch" | "onSettings" | "onFold">;
}) {
  const edges = deskShelfEdges(HARNESS_NOW, "UTC");
  const grouped = new Map(
    groupDeskThreadsByShelf(threads, edges, true).map((shelf) => [shelf.key, shelf.threads]),
  );
  const shelves: DeskRailShelf[] = DESK_SHELF_KEYS.map((key) => ({
    key,
    count: counts[key] ?? grouped.get(key)?.length ?? 0,
    threads: grouped.get(key) ?? [],
    paging: { ...READ_TO_END, ...paging[key] },
  }));
  const { handlers, ...rest } = props;

  return (
    <MemoryRouter initialEntries={["/desk"]}>
      <DeskRail
        place="today"
        view="chats"
        edges={edges}
        onSetPinned={vi.fn()}
        onRename={vi.fn()}
        onDownload={vi.fn()}
        onDelete={vi.fn()}
        agentsById={agents}
        activeThreadId={null}
        canWatch
        canDecide
        canTriage
        lensCounts={{ everything: 0, watchtower: 0, decisions: 0 }}
        triageWaiting={0}
        folded={false}
        mode="pinned"
        open={false}
        shelves={shelves}
        today={edges.today}
        noConversations={shelves.every((shelf) => shelf.threads.length === 0 && shelf.count === 0)}
        onLoadMore={onLoadMore}
        {...handlers}
        {...rest}
      />
      <LocationProbe />
    </MemoryRouter>
  );
}

/** Where the rail has sent the reader, for a place's link to be checked against. */
function LocationProbe() {
  return <span data-testid="location">{useLocation().pathname}</span>;
}

function renderRail(options: RailOptions = {}) {
  const handlers = {
    onSearch: vi.fn(),
    onSettings: vi.fn(),
    onFold: vi.fn(),
  };
  const onLoadMore = vi.fn<(shelf: DeskShelfKey) => void>();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const element = (next: RailOptions) => (
    <QueryClientProvider client={queryClient}>
      <RailHarness {...next} handlers={handlers} onLoadMore={onLoadMore} />
    </QueryClientProvider>
  );
  const view = render(element(options));

  return {
    ...view,
    handlers,
    onLoadMore,
    /** Renders the same rail again with new props, as the Desk does on every change. */
    update: (next: RailOptions) => view.rerender(element(next)),
  };
}

let restoreLayout = () => {};

beforeEach(() => {
  state.live = new Set();
  scheduled.byThread = new Map();
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  vi.useRealTimers();
  restoreLayout();
});

/**
 * The rail is the Desk's table of contents: the three places with what waits
 * at each, then every conversation shelved by when it was last touched, each
 * with a dot that says what it is waiting on.
 */
describe("DeskRail", () => {
  it("shelves conversations with pinned ones first", () => {
    const now = Math.floor(Date.now() / 1000);
    renderRail({
      threads: [
        thread({ id: "a", title: "Today's question" }),
        thread({ id: "b", title: "Old question", lastMessageAt: now - 3 * DAY }),
        thread({ id: "c", title: "Pinned question", pinned: true, lastMessageAt: now - 9 * DAY }),
      ],
    });

    const headings = [...document.querySelectorAll('[data-slot="rail-heading"]')].map(
      (node) => node.textContent,
    );
    expect(headings).toEqual(["Pinned", "Today", "Previous 7 days"]);
    expect(screen.getByText("Pinned question")).toBeInTheDocument();
  });

  it("says what each conversation waits on with its dot", () => {
    state.live = new Set(["live"]);
    renderRail({
      threads: [
        thread({ id: "live", title: "Working one" }),
        thread({
          id: "wait",
          title: "Waiting one",
          attention: { pendingDecisions: 2, lastTurnFailed: false, unread: false },
        }),
        thread({
          id: "fail",
          title: "Failed one",
          attention: { pendingDecisions: 0, lastTurnFailed: true, unread: false },
        }),
        thread({
          id: "new",
          title: "Unread one",
          attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true },
        }),
      ],
    });

    const dot = (title: string) =>
      screen
        .getByText(title)
        .closest('[data-slot="rail-row"]')
        ?.querySelector('[data-slot="rail-dot"]');
    expect(dot("Working one")).not.toHaveAttribute("data-on");
    expect(dot("Waiting one")).toHaveAttribute("data-on");
    expect(dot("Failed one")).not.toHaveAttribute("data-on");
    expect(dot("Unread one")).toHaveAttribute("data-on");
    expect(
      screen.getByText("Waiting one").closest('[data-slot="rail-row"]')?.getAttribute("title"),
    ).toMatch(/^Billing desk · Needs your approval · /);
  });

  it("says in words, with the agent's name, what a conversation needs or is doing", () => {
    state.live = new Set(["live"]);
    renderRail({
      threads: [
        thread({ id: "live", title: "Working one" }),
        thread({
          id: "wait",
          title: "Waiting one",
          attention: { pendingDecisions: 2, lastTurnFailed: false, unread: false },
        }),
        thread({
          id: "fail",
          title: "Failed one",
          attention: { pendingDecisions: 0, lastTurnFailed: true, unread: false },
        }),
        thread({
          id: "new",
          title: "Unread one",
          attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true },
        }),
        thread({ id: "quiet", title: "Quiet one" }),
      ],
    });

    const note = (title: string) =>
      screen
        .getByText(title)
        .closest('[data-slot="rail-row"]')
        ?.querySelector('[data-slot="rail-sub"]')?.textContent ?? null;
    expect(note("Working one")).toBe("Billing desk · Writing a reply");
    expect(note("Waiting one")).toBe("Billing desk · Needs your approval");
    expect(note("Failed one")).toBe("Billing desk · Last reply failed");
    // A new reply has its own dot; a conversation at rest names its agent.
    expect(note("Unread one")).toBe("Billing desk");
    expect(note("Quiet one")).toBe("Billing desk");
  });

  it("opens a conversation as its own page on a click, from Today as from anywhere", () => {
    useDeskMarginStore.getState().clearSession();
    renderRail({
      place: "today",
      threads: [thread({ id: "full", title: "Full one" })],
    });

    const row = screen.getByRole("link", { name: /^Full one/ });
    expect(row).toHaveAttribute("href", "/desk/t/full");
    // The click is the link's own: nothing takes it over to open the margin.
    fireEvent.click(row);
    expect(useDeskMarginStore.getState().margin).toBeNull();
  });

  it("lights the conversation that is open beside the page", () => {
    useDeskMarginStore.getState().clearSession();
    renderRail({
      place: "today",
      threads: [thread({ id: "beside", title: "Beside one" })],
    });

    act(() => useDeskMarginStore.getState().showThread("beside", null));

    expect(screen.getByText("Beside one").closest('[data-slot="rail-row"]')).toHaveAttribute(
      "aria-current",
      "page",
    );
    useDeskMarginStore.getState().clearSession();
  });

  it("makes each conversation a link to it that says which one is open, with nothing else in the row", () => {
    renderRail({
      place: "thread",
      activeThreadId: "open",
      threads: [
        thread({ id: "open", title: "Open one" }),
        thread({ id: "other", title: "Other one" }),
      ],
    });

    const open = screen.getByRole("link", { name: /^Open one/ });
    expect(open).toHaveAttribute("href", "/desk/t/open");
    expect(open).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: /^Other one/ })).not.toHaveAttribute("aria-current");
    expect(within(open).queryByRole("button")).toBeNull();
    expect(
      screen.queryByRole("button", { name: /Delete conversation|Pin conversation/ }),
    ).toBeNull();
  });

  it("does not call the open conversation unread", () => {
    renderRail({
      place: "thread",
      activeThreadId: "new",
      threads: [
        thread({
          id: "new",
          title: "Open one",
          attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true },
        }),
      ],
    });

    const row = screen.getByText("Open one").closest('[data-slot="rail-row"]');
    expect(row).toHaveAttribute("aria-current", "page");
    expect(row?.querySelector('[data-slot="rail-dot"]')).not.toHaveAttribute("data-on");
  });

  // The same rail, drawn again as the person moves: the row must follow the
  // conversation being opened and then read, not keep the dot it first drew.
  it("quiets a new reply's dot once its conversation is opened, and after it is read", () => {
    const unread = thread({
      id: "new",
      title: "Unread one",
      attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true },
    });
    const { update } = renderRail({ threads: [unread] });
    const row = () => screen.getByText("Unread one").closest('[data-slot="rail-row"]');

    expect(row()?.querySelector('[data-slot="rail-dot"]')).toHaveAttribute("data-on");

    update({ threads: [unread], place: "thread", activeThreadId: "new" });
    expect(row()).toHaveAttribute("aria-current", "page");
    expect(row()?.querySelector('[data-slot="rail-dot"]')).not.toHaveAttribute("data-on");

    const read = {
      ...unread,
      attention: { pendingDecisions: 0, lastTurnFailed: false, unread: false },
    };
    update({ threads: [read], place: "today", activeThreadId: null });
    expect(row()).not.toHaveAttribute("aria-current");
    expect(row()?.querySelector('[data-slot="rail-dot"]')).not.toHaveAttribute("data-on");
  });

  it("marks a reply that arrives later in a conversation already listed", () => {
    const seen = thread({
      id: "a",
      title: "Seen one",
      attention: { pendingDecisions: 0, lastTurnFailed: false, unread: false },
    });
    const { update } = renderRail({ threads: [seen] });
    const dot = () =>
      screen
        .getByText("Seen one")
        .closest('[data-slot="rail-row"]')
        ?.querySelector('[data-slot="rail-dot"]');

    expect(dot()).not.toHaveAttribute("data-on");
    update({
      threads: [
        { ...seen, attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true } },
      ],
    });
    expect(dot()).toHaveAttribute("data-on");
  });

  it("shows the way into cases on the conversations' rail, with what waits there", () => {
    renderRail({
      lensCounts: { everything: 19, watchtower: 11, decisions: 6 },
    });

    expect(screen.getByRole("link", { name: /Cases/ })).toHaveTextContent("19");
    expect(screen.getByRole("link", { name: /Cases/ })).toHaveAttribute("href", "/desk/needs");
    expect(screen.queryByRole("link", { name: /Watchtower/ })).toBeNull();
    expect(screen.getByRole("link", { name: "Scheduled" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Memory" })).toBeTruthy();
  });

  it("shows Needs you with what waits at each lens on the cases' rail", () => {
    renderRail({
      view: "cases",
      place: "needs",
      lensCounts: { everything: 19, watchtower: 11, decisions: 6 },
    });

    expect(screen.getByRole("link", { name: "Back to chats" })).toHaveAttribute("href", "/desk");
    expect(screen.queryByRole("link", { name: "Scheduled" })).toBeNull();

    expect(screen.getByRole("link", { name: /Needs you/ })).toHaveTextContent("19");
    expect(screen.getByRole("link", { name: /Watchtower/ })).toHaveTextContent("11");
    expect(screen.getByRole("link", { name: /Watchtower/ })).toHaveAttribute(
      "href",
      "/desk/needs?lens=watchtower",
    );
    expect(screen.getByRole("link", { name: /Decisions/ })).toHaveAttribute(
      "href",
      "/desk/needs?lens=decisions",
    );
    const decisions = screen.getByRole("link", { name: /Decisions/ });
    expect(decisions).toHaveTextContent("6");
  });

  it("opens Triage's room from either rail, with what waits on a person there", () => {
    const { unmount } = renderRail({ triageWaiting: 4 });

    expect(screen.getByRole("link", { name: /Triage/ })).toHaveAttribute("href", "/desk/triage");
    expect(screen.getByRole("link", { name: /Triage/ })).toHaveTextContent("4");
    unmount();

    renderRail({ view: "cases", place: "needs", triageWaiting: 4 });

    expect(screen.getByRole("link", { name: /Triage/ })).toHaveAttribute("href", "/desk/triage");
    expect(screen.getByRole("link", { name: /Triage/ })).toHaveTextContent("4");
  });

  it("gives the rail to Triage's folders while Triage is in front", () => {
    renderRail({
      view: "triage",
      place: "triage",
      lensCounts: { everything: 19, watchtower: 11, decisions: 6 },
    });

    expect(screen.getByRole("link", { name: "Back to chats" })).toHaveAttribute("href", "/desk");
    expect(screen.getByRole("link", { name: /Needs you/ })).toHaveTextContent("19");
    expect(screen.queryByRole("link", { name: /Watchtower/ })).toBeNull();
    expect(useDeskRailSlotStore.getState().slot).toBeInstanceOf(HTMLElement);
  });

  it("leads back from Triage to the page it was opened from, and names the view that page is in", () => {
    renderRail({
      view: "triage",
      place: "triage",
      triageBack: { view: "cases", to: "/desk/r/shipment/shp_1?item=wt_1" },
    });

    expect(screen.getByRole("link", { name: "Back to cases" })).toHaveAttribute(
      "href",
      "/desk/r/shipment/shp_1?item=wt_1",
    );
    expect(screen.queryByRole("link", { name: "Back to chats" })).toBeNull();
    expect(screen.queryByRole("link", { name: /Needs you/ })).toBeNull();
  });

  it("shows no count at a place where nothing waits", () => {
    renderRail({
      view: "cases",
      place: "needs",
      lensCounts: { everything: 0, watchtower: 0, decisions: 0 },
    });

    expect(
      screen.getByRole("link", { name: "Watchtower" }).querySelector('[data-slot="rail-count"]'),
    ).toBeNull();
    expect(
      screen.getByRole("link", { name: "Decisions" }).querySelector('[data-slot="rail-count"]'),
    ).toBeNull();
  });

  it("opens the scheduled requests from their place, and lights it while they are open", () => {
    renderRail();

    const scheduled = screen.getByRole("link", { name: "Scheduled" });
    expect(scheduled).not.toHaveAttribute("aria-current");
    fireEvent.click(scheduled);
    expect(screen.getByTestId("location")).toHaveTextContent("/desk/scheduled");
  });

  it("lights Scheduled, not Today, on the scheduled requests", () => {
    renderRail({ place: "scheduled" });

    expect(screen.getByRole("link", { name: "Scheduled" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Today" })).not.toHaveAttribute("aria-current");
  });

  it("leaves out the places a person may not open", () => {
    renderRail({ canWatch: false, canDecide: false });

    expect(screen.queryByRole("link", { name: /Watchtower/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Decisions/ })).not.toBeInTheDocument();
  });

  it("names who is signed in and opens search, the keys and settings", () => {
    const { handlers } = renderRail();

    expect(screen.getByText("Avery Lane")).toBeInTheDocument();
    expect(screen.getByText("AL")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back to Trenova" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Search or run a command" }));
    fireEvent.click(screen.getByRole("button", { name: "Desk settings" }));
    expect(handlers.onSearch).toHaveBeenCalled();
    expect(handlers.onSettings).toHaveBeenCalled();
  });
});

/**
 * The rail is 232px wide and folds away from the button left of the logo;
 * the logo leads back to Trenova.
 */
describe("DeskRail fold", () => {
  it("puts the fold first in the top row, then the logo that leads back to Trenova", () => {
    renderRail();

    const logo = screen.getByRole("link", { name: "Back to Trenova" });
    expect(logo).toHaveAttribute("href", "/");
    const top = logo.parentElement as HTMLElement;
    expect(top.firstElementChild).toBe(screen.getByRole("button", { name: "Hide the rail" }));
  });

  it("folds from the button left of the logo", () => {
    const { handlers } = renderRail();

    fireEvent.click(screen.getByRole("button", { name: "Hide the rail" }));

    expect(handlers.onFold).toHaveBeenCalledTimes(1);
  });

  it("opens a new conversation in Nova's margin", () => {
    useDeskMarginStore.getState().clearSession();
    renderRail();

    fireEvent.click(screen.getByRole("button", { name: "New conversation" }));

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: null, subject: null });
    useDeskMarginStore.getState().clearSession();
  });

  it("floats over the page as a card, out of reach until it is shown", () => {
    renderRail({ mode: "floating", open: false });

    const rail = screen.getByRole("complementary", { hidden: true });
    expect(rail).toHaveAttribute("data-mode", "floating");
    expect(rail).not.toHaveAttribute("data-open");
    expect(rail).toHaveAttribute("inert");
    expect(rail).toHaveClass("fixed", "rounded-xl");
  });

  it("is in reach once a floating rail is shown", () => {
    renderRail({ mode: "floating", open: true });

    const rail = screen.getByRole("complementary");
    expect(rail).toHaveAttribute("data-open");
    expect(rail).not.toHaveAttribute("inert");
  });

  it("sits beside the page when pinned, taking its width", () => {
    renderRail();

    const rail = screen.getByRole("complementary");
    expect(rail).toHaveAttribute("data-mode", "pinned");
    expect(rail).toHaveClass("w-58", "border-r");
    expect(rail).not.toHaveClass("rounded-xl");
  });

  it("takes no focus while folded", () => {
    renderRail({ folded: true });

    const rail = screen.getByRole("complementary", { hidden: true });
    expect(rail).toHaveAttribute("data-folded");
    expect(rail).toHaveAttribute("inert");
  });
});

describe("DeskRail rows", () => {
  // The name is changed from the top bar; a double-click on the rail is a
  // second click on the row, not an edit.
  it("does not turn a row into a field on double-click", () => {
    renderRail({ threads: [thread({ id: "a", title: "Old name" })] });

    fireEvent.doubleClick(screen.getByText("Old name"));

    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(screen.getByText("Old name")).toBeInTheDocument();
  });

  it("says where chats will show up when there are none", () => {
    renderRail();
    expect(screen.getByText(/Your chats will show up here/)).toBeInTheDocument();
  });
});

function many(count: number): AssistantThread[] {
  return Array.from({ length: count }, (_, index) =>
    thread({ id: `t${index}`, title: `Conversation ${index}` }),
  );
}

/**
 * Someone who has used the Desk for months has thousands of conversations.
 * Only the rows near the viewport are mounted, and each open shelf's next
 * page is read as the reader nears the place it arrives; a folded shelf reads
 * nothing.
 */
describe("DeskRail paging", () => {
  it("mounts only the conversations near the viewport", () => {
    renderRail({ threads: many(300) });

    const mounted = document.querySelectorAll('[data-slot="rail-row"]');
    expect(mounted.length).toBeGreaterThan(0);
    expect(mounted.length).toBeLessThan(40);
    expect(screen.getByText("Conversation 0")).toBeInTheDocument();
    expect(screen.queryByText("Conversation 299")).not.toBeInTheDocument();
  });

  it("asks for a shelf's next page when the reader is near where it arrives", () => {
    const { onLoadMore } = renderRail({ threads: many(3), paging: { today: { hasMore: true } } });

    expect(onLoadMore).toHaveBeenCalledWith("today");
    expect(onLoadMore).not.toHaveBeenCalledWith("yesterday");
  });

  it("does not ask while the reader is far from where it arrives", () => {
    const now = Math.floor(Date.now() / 1000);
    const { onLoadMore } = renderRail({
      threads: [...many(300), thread({ id: "y", lastMessageAt: now - DAY, createdAt: now - DAY })],
      paging: { yesterday: { hasMore: true } },
    });

    expect(onLoadMore).not.toHaveBeenCalled();
  });

  it("does not ask again while a page is loading, and shows that it is", () => {
    const { onLoadMore } = renderRail({
      threads: many(3),
      paging: { today: { hasMore: true, loading: true } },
    });

    expect(onLoadMore).not.toHaveBeenCalled();
    expect(screen.getByRole("status", { name: "Loading more..." })).toBeInTheDocument();
  });

  it("shows a shelf's first page loading under its heading", () => {
    renderRail({ counts: { today: 4 }, paging: { today: { loading: true } } });

    expect(
      screen.getByText("Today", { selector: '[data-slot="rail-heading"]' }),
    ).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Loading more..." })).toBeInTheDocument();
  });

  it("does not ask when a shelf has been read to its end", () => {
    const { onLoadMore } = renderRail({ threads: many(3) });

    expect(onLoadMore).not.toHaveBeenCalled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("offers to try again when a page could not be read", () => {
    const { onLoadMore } = renderRail({
      threads: many(3),
      paging: { today: { hasMore: true, failed: true } },
    });

    expect(onLoadMore).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't load more conversations");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(onLoadMore).toHaveBeenCalledTimes(1);
    expect(onLoadMore).toHaveBeenCalledWith("today");
  });
});

/**
 * The server counts every shelf, so the rail knows which shelves hold
 * anything without reading them. A heading names its shelf and nothing more:
 * the count decides whether it is drawn, not what it says.
 */
describe("DeskRail shelf counts", () => {
  it("leaves out a shelf counted empty", () => {
    renderRail({ threads: many(1), counts: { week: 0 } });

    expect(
      screen.queryByText("Previous 7 days", { selector: '[data-slot="rail-heading"]' }),
    ).not.toBeInTheDocument();
  });

  it("keeps the heading of a shelf that could not be counted", () => {
    renderRail({ threads: many(1), counts: { older: "unknown" } });

    expect(
      screen.getByText("Older", { selector: '[data-slot="rail-heading"]' }),
    ).toBeInTheDocument();
  });

  it("draws no heading for a shelf still being counted with nothing read", () => {
    renderRail({ threads: many(1), counts: { older: "counting" } });

    expect(
      screen.queryByText("Older", { selector: '[data-slot="rail-heading"]' }),
    ).not.toBeInTheDocument();
  });

  it("does not say there are no chats until every shelf is counted", () => {
    renderRail({ counts: { today: "counting" } });

    expect(screen.queryByText(/Your chats will show up here/)).not.toBeInTheDocument();
  });
});

describe("which rail a place belongs to", () => {
  it("keeps the cases' rail for a conversation asked on a record's case", () => {
    expect(railViewOf("thread", true)).toBe("cases");
    expect(railViewOf("thread", false)).toBe("chats");
    expect(railViewOf("thread")).toBe("chats");
  });

  it("is the cases' rail for case work and the conversations' everywhere else", () => {
    expect(railViewOf("case")).toBe("cases");
    expect(railViewOf("needs")).toBe("cases");
    expect(railViewOf("today", true)).toBe("chats");
    expect(railViewOf("memory")).toBe("chats");
  });
});
