import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  Notification,
  NotificationCounts,
  NotificationFeed,
  NotificationLens,
} from "@trenova/shared/types/notification";

const service = vi.hoisted(() => ({
  listNotifications: vi.fn(),
  getCounts: vi.fn(),
  getMyUnreadCount: vi.fn(),
  markRead: vi.fn(),
  markUnread: vi.fn(),
  dismiss: vi.fn(),
  dismissAll: vi.fn(),
  restore: vi.fn(),
  snooze: vi.fn(),
  wake: vi.fn(),
  markAllMineRead: vi.fn(),
}));

vi.mock("@trenova/shared/services/notification", () => ({ notificationService: service }));
vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: { error: vi.fn() } }));

import {
  lensOfNotification,
  patchNotification,
  useDismissAllNotifications,
  useNotificationAction,
  useNotificationCounts,
  useNotificationFeed,
  useSnoozeNotifications,
  useUnreadNotificationCount,
} from "@trenova/shared/hooks/use-notifications";

const NOW = 1_800_000_000;

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "notif_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: "usr_1",
    eventType: "watchtower.assigned",
    priority: "high",
    channel: "user",
    title: "Missed pickup",
    message: "The window closed and no arrival is recorded.",
    data: null,
    relatedEntities: null,
    source: "watchtower",
    readAt: null,
    dismissedAt: null,
    createdAt: NOW - 900,
    reason: "direct",
    occurrenceCount: 1,
    surfacesAt: NOW - 900,
    resolvedAt: null,
    resolution: "",
    ...overrides,
  };
}

function feedOf(results: Notification[]): NotificationFeed {
  return { results, totalCount: results.length, endCursor: null, hasNextPage: false };
}

/** A request that stays in flight until the test settles it, as the server would. */
function inFlight() {
  let settle: () => void = () => undefined;
  const promise = new Promise<void>((resolve) => {
    settle = resolve;
  });
  return { promise, settle };
}

function serve(
  feeds: Partial<Record<NotificationLens, Notification[]>>,
  counts: NotificationCounts,
) {
  service.listNotifications.mockImplementation(({ lens }: { lens: NotificationLens }) =>
    Promise.resolve(feedOf(feeds[lens] ?? [])),
  );
  service.getCounts.mockResolvedValue(counts);
}

function wrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

function usePanel(lens: NotificationLens) {
  return {
    feed: useNotificationFeed({ lens }, true),
    counts: useNotificationCounts(),
    done: useNotificationAction("dismiss"),
    later: useSnoozeNotifications(),
    allDone: useDismissAllNotifications(),
  };
}

async function renderPanel(lens: NotificationLens) {
  const view = renderHook(() => usePanel(lens), { wrapper: wrapper() });
  await waitFor(() => {
    expect(view.result.current.feed.data).toBeDefined();
    expect(view.result.current.counts.data).toBeDefined();
  });
  return view;
}

const ids = (result: { current: ReturnType<typeof usePanel> }) =>
  result.current.feed.data?.notifications.map((item) => item.id);

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now: NOW * 1000 });
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe("lensOfNotification", () => {
  it("puts an open notification that is the person's own in For you", () => {
    expect(lensOfNotification(notification(), NOW)).toBe("forYou");
  });

  it("puts an open notification that went to everyone who can act in Team", () => {
    expect(lensOfNotification(notification({ reason: "team" }), NOW)).toBe("team");
  });

  it("puts a notification that surfaces after now in Later, whatever its reason", () => {
    expect(lensOfNotification(notification({ surfacesAt: NOW + 1 }), NOW)).toBe("later");
    expect(lensOfNotification(notification({ reason: "team", surfacesAt: NOW + 60 }), NOW)).toBe(
      "later",
    );
  });

  it("puts a finished notification in Done even when it was set aside", () => {
    expect(
      lensOfNotification(notification({ dismissedAt: NOW - 5, surfacesAt: NOW + 3_600 }), NOW),
    ).toBe("done");
  });

  it("surfaces a notification at the second it was set aside until", () => {
    expect(lensOfNotification(notification({ surfacesAt: NOW }), NOW)).toBe("forYou");
  });
});

describe("patchNotification", () => {
  it("marks a notification done and read at once, keeping an earlier read time", () => {
    expect(patchNotification(notification(), { action: "dismiss" }, NOW)).toMatchObject({
      dismissedAt: NOW,
      readAt: NOW,
    });
    expect(
      patchNotification(notification({ readAt: NOW - 400 }), { action: "dismiss" }, NOW).readAt,
    ).toBe(NOW - 400);
  });

  it("sets a notification aside until a time and makes it unread for when it returns", () => {
    expect(
      patchNotification(
        notification({ readAt: NOW - 400 }),
        { action: "snooze", until: NOW + 3_600 },
        NOW,
      ),
    ).toMatchObject({ surfacesAt: NOW + 3_600, readAt: null });
  });

  it("brings a notification back now without changing whether it was read", () => {
    const woken = patchNotification(
      notification({ surfacesAt: NOW + 3_600, readAt: NOW - 400 }),
      { action: "wake" },
      NOW,
    );

    expect(woken).toMatchObject({ surfacesAt: NOW, readAt: NOW - 400 });
  });

  it("moves a finished notification back open, forgetting how it cleared itself", () => {
    const restored = patchNotification(
      notification({ dismissedAt: NOW - 60, resolvedAt: NOW - 60, resolution: "resolved" }),
      { action: "restore" },
      NOW,
    );

    expect(restored).toMatchObject({
      dismissedAt: null,
      resolvedAt: null,
      resolution: "",
      surfacesAt: NOW,
    });
  });
});

describe("the panel's writes", () => {
  it("takes a row marked done out of For you and off its counts before the server answers", async () => {
    serve(
      { forYou: [notification({ id: "a" }), notification({ id: "b", readAt: NOW - 60 })] },
      { forYou: 2, forYouUnread: 1, team: 4, teamUnread: 4, later: 0 },
    );
    const request = inFlight();
    service.dismiss.mockReturnValue(request.promise);
    const { result } = await renderPanel("forYou");

    act(() => result.current.done.mutate(["a"]));

    await waitFor(() => expect(ids(result)).toEqual(["b"]));
    expect(result.current.counts.data).toEqual({
      forYou: 1,
      forYouUnread: 0,
      team: 4,
      teamUnread: 4,
      later: 0,
    });
    expect(service.dismiss).toHaveBeenCalledWith(["a"], "all");

    await act(async () => request.settle());
  });

  it("moves a row set aside from its tab's count to Later's", async () => {
    serve(
      { team: [notification({ id: "a", reason: "team", readAt: NOW - 60 })] },
      { forYou: 0, forYouUnread: 0, team: 1, teamUnread: 0, later: 2 },
    );
    const request = inFlight();
    service.snooze.mockReturnValue(request.promise);
    const { result } = await renderPanel("team");

    act(() => result.current.later.mutate({ ids: ["a"], until: NOW + 3_600 }));

    await waitFor(() => expect(ids(result)).toEqual([]));
    expect(result.current.counts.data).toEqual({
      forYou: 0,
      forYouUnread: 0,
      team: 0,
      teamUnread: 0,
      later: 3,
    });
    expect(service.snooze).toHaveBeenCalledWith(["a"], NOW + 3_600);

    await act(async () => request.settle());
  });

  it("empties a tab marked all done, including the rows it has not loaded, and leaves the other alone", async () => {
    serve(
      {
        team: [
          notification({ id: "a", reason: "team" }),
          notification({ id: "b", reason: "team" }),
        ],
      },
      { forYou: 3, forYouUnread: 2, team: 40, teamUnread: 12, later: 1 },
    );
    const request = inFlight();
    service.dismissAll.mockReturnValue(request.promise);
    const { result } = await renderPanel("team");

    act(() => result.current.allDone.mutate("team"));

    await waitFor(() => expect(ids(result)).toEqual([]));
    expect(result.current.counts.data).toEqual({
      forYou: 3,
      forYouUnread: 2,
      team: 0,
      teamUnread: 0,
      later: 1,
    });
    expect(service.dismissAll).toHaveBeenCalledWith("team");

    await act(async () => request.settle());
  });

  it("puts the list back as the server has it when a write is refused", async () => {
    serve(
      { forYou: [notification({ id: "a" })] },
      { forYou: 1, forYouUnread: 1, team: 0, teamUnread: 0, later: 0 },
    );
    service.dismiss.mockRejectedValue(new Error("refused"));
    const { result } = await renderPanel("forYou");

    act(() => result.current.done.mutate(["a"]));

    await waitFor(() => expect(result.current.done.isError).toBe(true));
    await waitFor(() => expect(ids(result)).toEqual(["a"]));
    expect(result.current.counts.data?.forYou).toBe(1);
  });
});

describe("useUnreadNotificationCount", () => {
  it("counts for the bell only what is the person's own and unread", async () => {
    serve({}, { forYou: 9, forYouUnread: 6, team: 26, teamUnread: 26, later: 2 });

    const { result } = renderHook(() => useUnreadNotificationCount(), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.data).toBe(6));
    expect(service.getMyUnreadCount).not.toHaveBeenCalled();
  });

  it("counts every unread notification in the driver portal's one list", async () => {
    service.getMyUnreadCount.mockResolvedValue(4);

    const { result } = renderHook(() => useUnreadNotificationCount("mine"), { wrapper: wrapper() });

    await waitFor(() => expect(result.current.data).toBe(4));
    expect(service.getCounts).not.toHaveBeenCalled();
  });
});
