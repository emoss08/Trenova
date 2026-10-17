import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { stubLayout } from "@/test/layout";
import { api } from "@trenova/shared/lib/api";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
import { DeskRail, type DeskRailProps } from "../desk-rail";

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
  handlers: Pick<DeskRailProps, "onSearch" | "onSettings" | "onSetPinned" | "onDelete">;
}) {
  const collapsed = useDeskSettingsStore((store) => store.collapsedShelves);
  const edges = deskShelfEdges(HARNESS_NOW, "UTC");
  const grouped = new Map(
    groupDeskThreadsByShelf(threads, edges, true).map((shelf) => [shelf.key, shelf.threads]),
  );
  const shelves: DeskRailShelf[] = DESK_SHELF_KEYS.map((key) => ({
    key,
    collapsed: collapsed.includes(key),
    count: counts[key] ?? grouped.get(key)?.length ?? 0,
    threads: grouped.get(key) ?? [],
    paging: { ...READ_TO_END, ...paging[key] },
  }));
  const { handlers, ...rest } = props;

  return (
    <MemoryRouter initialEntries={["/desk"]}>
      <DeskRail
        place="today"
        agentsById={agents}
        activeThreadId={null}
        canWatch
        canDecide
        watchtowerCount={0}
        decisionsCount={0}
        decisionsWaitHere={false}
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

/** Presses delete on the row with this title, which asks in a dialog. */
function askToDelete(title: string) {
  const row = screen.getByText(title, { selector: ".dk-sb-t" }).closest(".dk-sb-c") as HTMLElement;
  fireEvent.click(within(row).getByRole("button", { name: "Delete conversation" }));
}

/** Answers the question asked before a conversation with schedules is deleted. */
function answerDelete(answer: "Delete" | "Cancel" = "Delete") {
  fireEvent.click(within(screen.getByRole("alertdialog")).getByRole("button", { name: answer }));
}

/** Where the rail has sent the reader, for a place's link to be checked against. */
function LocationProbe() {
  return <span data-testid="location">{useLocation().pathname}</span>;
}

function renderRail(options: RailOptions = {}) {
  const handlers = {
    onSearch: vi.fn(),
    onSettings: vi.fn(),
    onSetPinned: vi.fn(),
    onDelete: vi.fn(),
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
  useDeskSettingsStore.setState({ collapsedShelves: [] });
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

    const headings = [...document.querySelectorAll(".dk-sb-gh")].map((node) => node.textContent);
    expect(headings).toEqual(["Pinned", "Today", "Previous 7 days"]);
    expect(screen.getByText("Pinned question")).toBeInTheDocument();
  });

  it("folds a shelf from its heading and opens it again", () => {
    const now = Math.floor(Date.now() / 1000);
    renderRail({
      threads: [
        thread({ id: "a", title: "Today's question" }),
        thread({ id: "b", title: "Old question", lastMessageAt: now - 3 * DAY }),
      ],
    });

    fireEvent.click(screen.getByRole("button", { name: "Today", expanded: true }));

    expect(
      screen.getByRole("button", { name: "Today", expanded: false }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Today's question")).not.toBeInTheDocument();
    expect(screen.getByText("Old question")).toBeInTheDocument();
    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual(["today"]);

    fireEvent.click(screen.getByRole("button", { name: "Today", expanded: false }));

    expect(screen.getByText("Today's question")).toBeInTheDocument();
    expect(useDeskSettingsStore.getState().collapsedShelves).toEqual([]);
  });

  it("draws a shelf folded the way it was left", () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["today"] });
    renderRail({ threads: [thread({ id: "a", title: "Today's question" })] });

    expect(
      screen.getByRole("button", { name: "Today", expanded: false }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Today's question")).not.toBeInTheDocument();
  });

  it("keeps the open conversation on a folded shelf", () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["today"] });
    renderRail({
      place: "thread",
      activeThreadId: "a",
      threads: [
        thread({ id: "a", title: "Open question" }),
        thread({ id: "b", title: "Other question" }),
      ],
    });

    expect(screen.getByText("Open question")).toBeInTheDocument();
    expect(screen.queryByText("Other question")).not.toBeInTheDocument();
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
      screen.getByText(title).closest(".dk-sb-c")?.querySelector(".dk-sb-dot");
    expect(dot("Working one")).toHaveClass("dk-s-work");
    expect(dot("Waiting one")).toHaveClass("dk-s-wait");
    expect(dot("Failed one")).toHaveClass("dk-s-error");
    expect(dot("Unread one")).toHaveClass("dk-s-new");
    expect(screen.getByText("Waiting one").closest(".dk-sb-c")?.getAttribute("title")).toMatch(
      /^Billing desk · Needs your approval · /,
    );
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

    const row = screen.getByText("Open one").closest(".dk-sb-c");
    expect(row).toHaveClass("dk-on");
    expect(row?.querySelector(".dk-sb-dot")).not.toHaveClass("dk-s-new");
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
    const row = () => screen.getByText("Unread one").closest(".dk-sb-c");

    expect(row()?.querySelector(".dk-sb-dot")).toHaveClass("dk-s-new");

    update({ threads: [unread], place: "thread", activeThreadId: "new" });
    expect(row()).toHaveClass("dk-on");
    expect(row()?.querySelector(".dk-sb-dot")).not.toHaveClass("dk-s-new");

    const read = {
      ...unread,
      attention: { pendingDecisions: 0, lastTurnFailed: false, unread: false },
    };
    update({ threads: [read], place: "today", activeThreadId: null });
    expect(row()).not.toHaveClass("dk-on");
    expect(row()?.querySelector(".dk-sb-dot")).not.toHaveClass("dk-s-new");
  });

  it("marks a reply that arrives later in a conversation already listed", () => {
    const seen = thread({
      id: "a",
      title: "Seen one",
      attention: { pendingDecisions: 0, lastTurnFailed: false, unread: false },
    });
    const { update } = renderRail({ threads: [seen] });
    const dot = () => screen.getByText("Seen one").closest(".dk-sb-c")?.querySelector(".dk-sb-dot");

    expect(dot()).not.toHaveClass("dk-s-new");
    update({
      threads: [
        { ...seen, attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true } },
      ],
    });
    expect(dot()).toHaveClass("dk-s-new");
  });

  it("pins in one click and deletes a conversation with no schedules without asking", async () => {
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers } = renderRail({ threads: [subject] });

    fireEvent.click(screen.getByRole("button", { name: "Pin conversation" }));
    expect(handlers.onSetPinned).toHaveBeenCalledWith(subject, true);

    askToDelete("Blocked invoices");

    await waitFor(() => expect(handlers.onDelete).toHaveBeenCalledWith(subject));
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("asks before deleting a conversation with schedules, and lists them", async () => {
    scheduled.byThread.set("a", 2);
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers } = renderRail({ threads: [subject] });

    askToDelete("Blocked invoices");

    const card = within(await screen.findByRole("alertdialog"));
    expect(card.getByText("Delete “Blocked invoices”?")).toBeVisible();
    expect(card.getByText("Request 1")).toBeVisible();
    expect(card.getByText("Request 2")).toBeVisible();
    expect(handlers.onDelete).not.toHaveBeenCalled();

    answerDelete();

    await waitFor(() => expect(handlers.onDelete).toHaveBeenCalledWith(subject));
  });

  it("keeps a conversation with schedules when the question is cancelled", async () => {
    scheduled.byThread.set("a", 1);
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers } = renderRail({ threads: [subject] });

    askToDelete("Blocked invoices");
    await screen.findByRole("alertdialog");
    answerDelete("Cancel");

    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
    expect(handlers.onDelete).not.toHaveBeenCalled();
    expect(screen.getByText("Blocked invoices").closest(".dk-sb-c")).not.toHaveClass("dk-out");
  });

  // The rail is given the conversation with its new pin a task after the
  // click; the button must not show the old state in between, and a second
  // click must undo the first rather than ask for it again.
  it("shows a pin from the click, and a second click unpins before the rail is told", () => {
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers } = renderRail({ threads: [subject] });

    fireEvent.click(screen.getByRole("button", { name: "Pin conversation" }));
    const pin = screen.getByRole("button", { name: "Unpin conversation" });
    expect(pin).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(pin);

    expect(handlers.onSetPinned).toHaveBeenNthCalledWith(1, subject, true);
    expect(handlers.onSetPinned).toHaveBeenNthCalledWith(2, subject, false);
    expect(screen.getByRole("button", { name: "Pin conversation" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  it("shows the pin the conversation carries once the rail is told", () => {
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { update } = renderRail({ threads: [subject] });

    fireEvent.click(screen.getByRole("button", { name: "Pin conversation" }));
    update({ threads: [{ ...subject, pinned: true }] });
    update({ threads: [subject] });

    expect(screen.getByRole("button", { name: "Pin conversation" })).toHaveAttribute(
      "aria-pressed",
      "false",
    );
  });

  // The rail is given shelves without a deleted conversation a task after the
  // delete is sent; the row must not come back in between.
  it("keeps a deleted row slid out until the shelves drop it", async () => {
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const other = thread({ id: "b", title: "Late loads" });
    const { handlers, update } = renderRail({ threads: [subject, other] });
    const row = () => screen.queryByText("Blocked invoices")?.closest(".dk-sb-c");

    askToDelete("Blocked invoices");
    await waitFor(() => expect(handlers.onDelete).toHaveBeenCalledWith(subject));
    update({ threads: [subject, other] });

    expect(row()).toHaveClass("dk-out");

    update({ threads: [other] });

    expect(row()).toBeUndefined();
  });

  it("shows a deleted row again when a refused delete brings it back", async () => {
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers, update } = renderRail({ threads: [subject] });

    askToDelete("Blocked invoices");
    await waitFor(() => expect(handlers.onDelete).toHaveBeenCalledWith(subject));
    update({ threads: [] });
    update({ threads: [subject] });

    expect(screen.getByText("Blocked invoices").closest(".dk-sb-c")).not.toHaveClass("dk-out");
  });

  it("shows the places with what waits at each", () => {
    renderRail({ watchtowerCount: 11, decisionsCount: 6, decisionsWaitHere: true });

    expect(screen.getByRole("button", { name: /Watchtower/ })).toHaveTextContent("11");
    const decisions = screen.getByRole("button", { name: /Decisions/ });
    expect(decisions).toHaveTextContent("6");
    expect(decisions.querySelector(".dk-sb-ct")).toHaveClass("dk-w");
  });

  it("opens the scheduled requests from their place, and lights it while they are open", () => {
    renderRail();

    const scheduled = screen.getByRole("button", { name: "Scheduled" });
    expect(scheduled).not.toHaveClass("text-dsk-fg");
    fireEvent.click(scheduled);
    expect(screen.getByTestId("location")).toHaveTextContent("/desk/scheduled");
  });

  it("lights Scheduled, not Today, on the scheduled requests", () => {
    renderRail({ place: "scheduled" });

    expect(screen.getByRole("button", { name: "Scheduled" })).toHaveAttribute("data-k", "n:sched");
    expect(screen.getByRole("button", { name: "Scheduled" })).toHaveClass("text-dsk-fg");
    expect(document.querySelector('[data-k="n:today"]')).not.toHaveClass("text-dsk-fg");
  });

  it("leaves out the places a person may not open", () => {
    renderRail({ canWatch: false, canDecide: false });

    expect(screen.queryByRole("button", { name: /Watchtower/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Decisions/ })).not.toBeInTheDocument();
  });

  it("names who is signed in and opens search and settings", () => {
    const { handlers } = renderRail();

    expect(screen.getByText("Avery Lane")).toBeInTheDocument();
    expect(screen.getByText("AL")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    fireEvent.click(screen.getByRole("button", { name: "Desk settings" }));
    expect(handlers.onSearch).toHaveBeenCalled();
    expect(handlers.onSettings).toHaveBeenCalled();
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

    const mounted = document.querySelectorAll(".dk-sb-c");
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

    expect(screen.getByRole("button", { name: "Today", expanded: true })).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Loading more..." })).toBeInTheDocument();
  });

  it("does not ask when a shelf has been read to its end", () => {
    const { onLoadMore } = renderRail({ threads: many(3) });

    expect(onLoadMore).not.toHaveBeenCalled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("reads nothing of a folded shelf, and shows nothing loading under it", () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["today"] });
    const { onLoadMore } = renderRail({
      counts: { today: 120 },
      paging: { today: { hasMore: true, loading: true } },
    });

    expect(onLoadMore).not.toHaveBeenCalled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  // A failed page is not asked for again on every render, which would hammer
  // a server that is already struggling; the person retries it.
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
  it("draws a folded shelf's heading by its name alone", () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["month"] });
    renderRail({ counts: { month: 42 } });

    const heading = screen.getByRole("button", { name: "Previous 30 days", expanded: false });
    expect(heading).toHaveTextContent(/^Previous 30 days$/);
    expect(heading.querySelector(".dk-sb-ct")).toBeNull();
  });

  it("leaves out a shelf counted empty", () => {
    renderRail({ threads: many(1), counts: { week: 0 } });

    expect(screen.queryByRole("button", { name: /Previous 7 days/ })).not.toBeInTheDocument();
  });

  it("keeps the heading of a shelf that could not be counted, so it can be opened", () => {
    useDeskSettingsStore.setState({ collapsedShelves: ["older"] });
    renderRail({ threads: many(1), counts: { older: "unknown" } });

    expect(screen.getByRole("button", { name: "Older", expanded: false })).toBeInTheDocument();
  });

  it("draws no heading for a shelf still being counted with nothing read", () => {
    renderRail({ threads: many(1), counts: { older: "counting" } });

    expect(screen.queryByRole("button", { name: /Older/ })).not.toBeInTheDocument();
  });

  it("does not say there are no chats until every shelf is counted", () => {
    renderRail({ counts: { today: "counting" } });

    expect(screen.queryByText(/Your chats will show up here/)).not.toBeInTheDocument();
  });
});
