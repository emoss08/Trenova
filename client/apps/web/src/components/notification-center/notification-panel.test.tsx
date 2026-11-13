import type {
  Notification,
  NotificationCounts,
  NotificationLens,
} from "@trenova/shared/types/notification";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRef, type ReactNode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { stubLayout } from "@/test/layout";
import NotificationPanel, { type NotificationPanelKeyHandler } from "./notification-panel";

type Write = { mutate: ReturnType<typeof vi.fn>; mutateAsync: ReturnType<typeof vi.fn> };

const hooks = vi.hoisted(() => ({
  feeds: {} as Partial<Record<string, unknown[]>>,
  feedState: {} as Record<string, unknown>,
  asked: [] as string[],
  counts: undefined as unknown,
  writes: {} as Record<string, { mutate: unknown; mutateAsync: unknown }>,
  dismissAll: { mutate: vi.fn(), isPending: false },
  refetch: vi.fn(),
}));
const requestGraphQL = vi.hoisted(() => vi.fn());
const toast = vi.hoisted(() => ({ error: vi.fn(), success: vi.fn() }));

function write(name: string): Write {
  hooks.writes[name] ??= { mutate: vi.fn(), mutateAsync: vi.fn(() => Promise.resolve()) };
  return hooks.writes[name] as Write;
}

vi.mock("@trenova/shared/hooks/use-notifications", () => ({
  useNotificationFeed: ({ lens }: { lens: NotificationLens }) => {
    hooks.asked.push(lens);
    return {
      data: { notifications: hooks.feeds[lens] ?? [], totalCount: 0 },
      isLoading: false,
      isError: false,
      refetch: hooks.refetch,
      fetchNextPage: vi.fn(),
      hasNextPage: false,
      isFetchingNextPage: false,
      ...hooks.feedState,
    };
  },
  useNotificationCounts: () => ({ data: hooks.counts }),
  useDismissAllNotifications: () => hooks.dismissAll,
  useNotificationAction: (action: string) => write(action),
  useSnoozeNotifications: () => write("snooze"),
  useReplyToMention: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast }));
vi.mock("@/components/resolved-user-avatar", () => ({ ResolvedUserAvatar: () => null }));
// The panel is hosted by a sheet, and only takes its title from it. A dialog's focus
// management reads the test DOM, which has no layout, as if every row were hidden and pulls
// the focus back to the first control it can see, so the host here is a plain element.
vi.mock("@trenova/shared/components/ui/sheet", () => ({
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

const NOW = Math.floor(Date.now() / 1000);

function notification(id: string, overrides: Partial<Notification> = {}): Notification {
  return {
    id,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: "usr_1",
    eventType: "watchtower.assigned",
    priority: "high",
    channel: "user",
    title: `Exception ${id}`,
    message: "The window closed and no arrival is recorded.",
    data: { itemId: `wti_${id}`, link: `/desk/needs?item=${id}` },
    relatedEntities: null,
    source: "watchtower",
    readAt: null,
    dismissedAt: null,
    createdAt: NOW - 600,
    reason: "direct",
    occurrenceCount: 1,
    surfacesAt: NOW - 600,
    resolvedAt: null,
    resolution: "",
    ...overrides,
  };
}

const COUNTS: NotificationCounts = {
  forYou: 9,
  forYouUnread: 6,
  team: 26,
  teamUnread: 20,
  later: 2,
};

function Where() {
  const location = useLocation();
  return <output data-testid="where">{location.pathname + location.search}</output>;
}

function renderPanel() {
  const onClose = vi.fn();
  // The host hands every key pressed inside it to the panel, as NotificationSheet does.
  const keys = createRef<NotificationPanelKeyHandler>();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter initialEntries={["/home"]}>
        <div onKeyDown={(event) => keys.current?.(event)}>
          <NotificationPanel open onClose={onClose} keys={keys} />
        </div>
        <Where />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { onClose };
}

const rowOf = (id: string) => screen.getByRole("button", { name: new RegExp(`Exception ${id}$`) });

// The list mounts only the rows near its viewport, so it needs a viewport with a height.
let restoreLayout = () => {};

afterEach(() => {
  restoreLayout();
});

beforeEach(() => {
  restoreLayout = stubLayout(900);
  hooks.feeds = { forYou: [notification("a"), notification("b"), notification("c")] };
  hooks.feedState = {};
  hooks.asked = [];
  hooks.counts = COUNTS;
  hooks.writes = {};
  hooks.dismissAll = { mutate: vi.fn(), isPending: false };
  hooks.refetch.mockReset();
  requestGraphQL.mockReset();
  toast.error.mockReset();
});

describe("NotificationPanel — its header", () => {
  it("counts each part on its tab, and says nothing else about what waits", () => {
    renderPanel();

    expect(screen.queryByText(/for your team/)).toBeNull();
    expect(screen.getByRole("tab", { name: "For you 9" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Team 26" })).toBeVisible();
    expect(screen.getByRole("tab", { name: "Later 2" })).toBeVisible();
    expect(screen.getByRole("tab", { name: "Done" })).toBeVisible();
  });

  it("opens the notification settings and closes the panel", () => {
    const { onClose } = renderPanel();

    fireEvent.click(screen.getByRole("button", { name: "Notification settings" }));

    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("where")).toHaveTextContent("/settings?section=notifications");
  });
});

describe("NotificationPanel — its parts", () => {
  it("reads only the part that is open", () => {
    hooks.feeds.team = [notification("t", { reason: "team" })];
    renderPanel();
    expect(new Set(hooks.asked)).toEqual(new Set(["forYou"]));

    fireEvent.click(screen.getByRole("tab", { name: "Team 26" }));

    expect(hooks.asked.at(-1)).toBe("team");
    expect(rowOf("t")).toBeVisible();
    expect(screen.queryByRole("button", { name: /Exception a$/ })).toBeNull();
  });

  it("groups rows under the day they surfaced", () => {
    hooks.feeds.forYou = [
      notification("a"),
      notification("old", { surfacesAt: NOW - 20 * 86_400 }),
    ];
    renderPanel();

    expect(screen.getByText("Today")).toBeVisible();
    expect(screen.getByText("Older")).toBeVisible();
  });

  it.each([
    ["forYou", "For you 0", "Nothing is waiting on you"],
    ["team", "Team 0", "Your team is clear"],
    ["later", "Later 0", "Nothing set aside"],
    ["done", "Done", "Nothing done yet"],
  ])("draws the %s part empty in its own words", (_lens, tab, title) => {
    hooks.feeds = {};
    hooks.counts = { forYou: 0, forYouUnread: 0, team: 0, teamUnread: 0, later: 0 };
    renderPanel();

    fireEvent.click(screen.getByRole("tab", { name: tab }));

    expect(screen.getByRole("heading", { name: title })).toBeVisible();
  });

  it("says a part could not be loaded and reads it again when asked", () => {
    hooks.feedState = { isError: true, data: undefined };
    renderPanel();

    expect(screen.getByText("Notifications couldn't be loaded.")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(hooks.refetch).toHaveBeenCalledTimes(1);
  });
});

describe("NotificationPanel — marking a whole part done", () => {
  it("marks the open part done", () => {
    renderPanel();

    fireEvent.click(screen.getByRole("button", { name: "Mark all done" }));

    expect(hooks.dismissAll.mutate).toHaveBeenCalledWith("forYou");
  });

  it("offers it for Team, and not for what is set aside or already done", () => {
    renderPanel();

    fireEvent.click(screen.getByRole("tab", { name: "Team 26" }));
    expect(screen.getByRole("button", { name: "Mark all done" })).toBeVisible();

    fireEvent.click(screen.getByRole("tab", { name: "Later 2" }));
    expect(screen.queryByRole("button", { name: "Mark all done" })).toBeNull();

    fireEvent.click(screen.getByRole("tab", { name: "Done" }));
    expect(screen.queryByRole("button", { name: "Mark all done" })).toBeNull();
  });

  it("does not offer it for a part with nothing in it", () => {
    hooks.counts = { ...COUNTS, forYou: 0, forYouUnread: 0 };
    hooks.feeds = {};
    renderPanel();

    expect(screen.queryByRole("button", { name: "Mark all done" })).toBeNull();
  });
});

describe("NotificationPanel — the keyboard", () => {
  it("moves down and up the list with J and K", async () => {
    renderPanel();
    rowOf("a").focus();

    fireEvent.keyDown(rowOf("a"), { key: "j" });
    await waitFor(() => expect(rowOf("b")).toHaveFocus());

    fireEvent.keyDown(rowOf("b"), { key: "k" });
    await waitFor(() => expect(rowOf("a")).toHaveFocus());
  });

  it("starts at the first row when no row has the focus", async () => {
    renderPanel();

    fireEvent.keyDown(screen.getByRole("tab", { name: "For you 9" }), { key: "j" });

    await waitFor(() => expect(rowOf("a")).toHaveFocus());
  });

  it("finishes the focused row with E and moves on to the next", async () => {
    renderPanel();
    rowOf("a").focus();

    fireEvent.keyDown(rowOf("a"), { key: "e" });

    await waitFor(() => expect(write("dismiss").mutateAsync).toHaveBeenCalledWith(["a"]));
    await waitFor(() => expect(rowOf("b")).toHaveFocus());
  });

  it("leaves keys alone while the person is typing a reply", () => {
    hooks.feeds.forYou = [
      notification("a", {
        eventType: "shipment_comment_mention",
        data: { authorName: "Dana Whitfield" },
        relatedEntities: { shipmentId: "shp_1" },
      }),
      notification("b"),
    ];
    renderPanel();

    fireEvent.keyDown(screen.getByRole("textbox", { name: "Reply to Dana Whitfield…" }), {
      key: "e",
    });

    expect(write("dismiss").mutateAsync).not.toHaveBeenCalled();
    expect(write("dismiss").mutate).not.toHaveBeenCalled();
  });

  it("does not finish a row that is already done", () => {
    hooks.feeds.done = [notification("d", { dismissedAt: NOW - 60, readAt: NOW - 60 })];
    renderPanel();
    fireEvent.click(screen.getByRole("tab", { name: "Done" }));

    fireEvent.keyDown(rowOf("d"), { key: "e" });

    expect(write("dismiss").mutateAsync).not.toHaveBeenCalled();
  });
});

describe("NotificationPanel — a row's writes", () => {
  it("opens the record, marks the row read and closes the panel", () => {
    const { onClose } = renderPanel();

    fireEvent.click(rowOf("a"));

    expect(write("read").mutate).toHaveBeenCalledWith(["a"]);
    expect(write("dismiss").mutate).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("where")).toHaveTextContent("/desk/needs?item=a");
  });

  it("finishes a row whose opening is all it asked for", () => {
    hooks.feeds.forYou = [
      notification("a", {
        eventType: "assistant_reply_ready",
        data: { threadId: "athr_1", status: "Completed", link: "/desk/c/athr_1" },
      }),
    ];
    renderPanel();

    fireEvent.click(screen.getByRole("button", { name: "Read the reply" }));

    expect(write("dismiss").mutate).toHaveBeenCalledWith(["a"]);
    expect(write("read").mutate).not.toHaveBeenCalled();
    expect(screen.getByTestId("where")).toHaveTextContent("/desk/c/athr_1");
  });

  it("acknowledges an exception through the Desk's own operation, then finishes the row", async () => {
    requestGraphQL.mockResolvedValue({ acknowledgeWatchtowerItem: { id: "wti_a" } });
    const { onClose } = renderPanel();

    fireEvent.click(screen.getAllByRole("button", { name: "Acknowledge" })[0]);

    await waitFor(() => expect(write("dismiss").mutateAsync).toHaveBeenCalledWith(["a"]));
    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AcknowledgeWatchtowerItem",
        variables: { id: "wti_a" },
      }),
    );
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps the row and says why when the exception could not be acknowledged", async () => {
    requestGraphQL.mockRejectedValue(new Error("You cannot acknowledge this exception"));
    renderPanel();
    const acknowledge = screen.getAllByRole("button", { name: "Acknowledge" })[0];

    fireEvent.click(acknowledge);

    await waitFor(() => expect(toast.error).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(acknowledge).toBeEnabled());
    expect(write("dismiss").mutateAsync).not.toHaveBeenCalled();
    expect(rowOf("a")).toBeVisible();
  });

  it("brings a row set aside back now", async () => {
    hooks.feeds.later = [notification("l", { surfacesAt: NOW + 3_600 })];
    renderPanel();
    fireEvent.click(screen.getByRole("tab", { name: "Later 2" }));

    fireEvent.click(screen.getByRole("button", { name: "Bring back now" }));

    await waitFor(() => expect(write("wake").mutateAsync).toHaveBeenCalledWith(["l"]));
  });

  it("moves a finished row back", async () => {
    hooks.feeds.done = [notification("d", { dismissedAt: NOW - 60, readAt: NOW - 60 })];
    renderPanel();
    fireEvent.click(screen.getByRole("tab", { name: "Done" }));

    fireEvent.click(screen.getByRole("button", { name: "Move back" }));

    await waitFor(() => expect(write("restore").mutateAsync).toHaveBeenCalledWith(["d"]));
  });
});
