import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import NotificationList from "../notification-list";
import { NotificationListSkeleton } from "../notification-list-skeleton";

const mocks = vi.hoisted(() => ({
  listNotifications: vi.fn(),
  getUnreadCount: vi.fn(),
}));

vi.mock("@trenova/shared/services/notification", () => ({
  notificationService: {
    listNotifications: mocks.listNotifications,
    getUnreadCount: mocks.getUnreadCount,
  },
}));

vi.mock("@/services/api", () => ({ apiService: { notificationService: {} } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("NotificationList", () => {
  // The feed is read before anything is listed; the tab draws the list's rows
  // meanwhile, the same outline the tab draws while its code arrives.
  it("outlines the notification rows while the feed is read", () => {
    const pending = new Promise<never>(() => {});
    mocks.listNotifications.mockReturnValue(pending);
    mocks.getUnreadCount.mockReturnValue(pending);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <NotificationList />
      </QueryClientProvider>,
    );

    expect(screen.getByLabelText("Loading notifications")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByText("No notifications yet")).toBeNull();
  });

  it("carries no text in its outline", () => {
    render(<NotificationListSkeleton />);

    expect(screen.getByLabelText("Loading notifications")).toHaveTextContent("");
  });
});
