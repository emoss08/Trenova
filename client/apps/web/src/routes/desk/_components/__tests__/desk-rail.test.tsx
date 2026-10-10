import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { stubLayout } from "@/test/layout";
import type { AssistantThread } from "@/types/assistant";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskRail, type DeskRailProps } from "../desk-rail";

const DAY = 24 * 60 * 60;

const state = vi.hoisted(() => ({ live: new Set<string>() }));

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

function renderRail(props: Partial<DeskRailProps> = {}) {
  const handlers = {
    onSearch: vi.fn(),
    onSettings: vi.fn(),
    onTogglePin: vi.fn(),
    onDelete: vi.fn(),
  };
  const element = (next: Partial<DeskRailProps>) => (
    <MemoryRouter initialEntries={["/desk"]}>
      <DeskRail
        place="today"
        threads={[]}
        agentsById={agents}
        activeThreadId={null}
        canWatch
        canDecide
        watchtowerCount={0}
        decisionsCount={0}
        decisionsWaitHere={false}
        {...handlers}
        {...next}
      />
    </MemoryRouter>
  );
  const view = render(element(props));

  return {
    ...view,
    handlers,
    /** Renders the same rail again with new props, as the Desk does on every change. */
    update: (next: Partial<DeskRailProps>) => view.rerender(element(next)),
  };
}

let restoreLayout = () => {};

beforeEach(() => {
  state.live = new Set();
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
    const dot = () =>
      screen.getByText("Seen one").closest(".dk-sb-c")?.querySelector(".dk-sb-dot");

    expect(dot()).not.toHaveClass("dk-s-new");
    update({
      threads: [
        { ...seen, attention: { pendingDecisions: 0, lastTurnFailed: false, unread: true } },
      ],
    });
    expect(dot()).toHaveClass("dk-s-new");
  });

  it("pins in one click and deletes only after asking again", () => {
    vi.useFakeTimers();
    const subject = thread({ id: "a", title: "Blocked invoices" });
    const { handlers } = renderRail({ threads: [subject] });

    fireEvent.click(screen.getByRole("button", { name: "Pin conversation" }));
    expect(handlers.onTogglePin).toHaveBeenCalledWith(subject);

    fireEvent.click(screen.getByRole("button", { name: "Delete conversation" }));
    expect(handlers.onDelete).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(handlers.onDelete).toHaveBeenCalledWith(subject);
  });

  it("drops the delete question when the pointer leaves the row", () => {
    renderRail({ threads: [thread({ id: "a", title: "Blocked invoices" })] });

    fireEvent.click(screen.getByRole("button", { name: "Delete conversation" }));
    expect(screen.getByRole("button", { name: "Delete" })).toBeInTheDocument();
    fireEvent.mouseLeave(screen.getByText("Blocked invoices").closest(".dk-sb-c") as Element);
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
  });

  it("shows the places with what waits at each", () => {
    renderRail({ watchtowerCount: 11, decisionsCount: 6, decisionsWaitHere: true });

    expect(screen.getByRole("button", { name: /Watchtower/ })).toHaveTextContent("11");
    const decisions = screen.getByRole("button", { name: /Decisions/ });
    expect(decisions).toHaveTextContent("6");
    expect(decisions.querySelector(".dk-sb-ct")).toHaveClass("dk-w");
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

function paging(overrides: Partial<NonNullable<DeskRailProps["paging"]>> = {}) {
  return {
    hasMore: true,
    loadingMore: false,
    failed: false,
    loadMore: vi.fn(),
    ...overrides,
  };
}

function many(count: number): AssistantThread[] {
  return Array.from({ length: count }, (_, index) =>
    thread({ id: `t${index}`, title: `Conversation ${index}` }),
  );
}

/**
 * Someone who has used the Desk for months has thousands of conversations.
 * Only the rows near the viewport are mounted, and the next page is read as
 * the reader nears the place it arrives.
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

  it("asks for the next page when the reader is near where it arrives", () => {
    const more = paging();
    renderRail({ threads: many(3), paging: more });

    expect(more.loadMore).toHaveBeenCalled();
  });

  it("does not ask while the reader is far from the end", () => {
    const more = paging();
    renderRail({ threads: many(300), paging: more });

    expect(more.loadMore).not.toHaveBeenCalled();
  });

  it("does not ask again while a page is loading, and shows that it is", () => {
    const more = paging({ loadingMore: true });
    renderRail({ threads: many(3), paging: more });

    expect(more.loadMore).not.toHaveBeenCalled();
    expect(screen.getByRole("status", { name: "Loading more..." })).toBeInTheDocument();
  });

  it("does not ask when every conversation has been read", () => {
    const more = paging({ hasMore: false });
    renderRail({ threads: many(3), paging: more });

    expect(more.loadMore).not.toHaveBeenCalled();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  // A failed page is not asked for again on every render, which would hammer
  // a server that is already struggling; the person retries it.
  it("offers to try again when a page could not be read", () => {
    const more = paging({ failed: true });
    renderRail({ threads: many(3), paging: more });

    expect(more.loadMore).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn't load more conversations");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(more.loadMore).toHaveBeenCalledTimes(1);
  });
});
