import { snoozePresets } from "@/lib/snooze-presets";
import {
  createDropdownMenuHandle,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import type { Notification } from "@trenova/shared/types/notification";
import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  laterTriggerId,
  NotificationLaterMenu,
  NotificationMoreMenu,
} from "./notification-row-menus";
import type { NotificationRowHandlers } from "./use-notification-row-handlers";

vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "America/Chicago" }));

// A Friday at 10:00 in Chicago: later today, tomorrow and Monday are all still ahead.
const NOW = Date.UTC(2026, 10, 13, 16, 0, 0) / 1000;

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "notif_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: "usr_1",
    eventType: "tender_accepted",
    priority: "low",
    channel: "user",
    title: "Tender accepted",
    message: "Blue Ridge Freight accepted the tender",
    data: { link: "/dispatch/console?move=smv_1" },
    relatedEntities: null,
    source: "tender",
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

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now: NOW * 1000 });
});

afterEach(() => {
  vi.useRealTimers();
});

function openLaterMenu(row: Notification) {
  const handle = createDropdownMenuHandle<Notification>();
  const onLater = vi.fn();
  render(
    <>
      <DropdownMenuTrigger id={laterTriggerId(row.id)} handle={handle} payload={row}>
        Later
      </DropdownMenuTrigger>
      <NotificationLaterMenu handle={handle} onLater={onLater} nextRow={() => null} />
    </>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Later" }));

  return { onLater };
}

function openMoreMenu(row: Notification) {
  const handle = createDropdownMenuHandle<Notification>();
  const handlers = {
    read: vi.fn(),
    unread: vi.fn(),
    turnOff: vi.fn(),
    copyLink: vi.fn(),
  };
  render(
    <>
      <DropdownMenuTrigger handle={handle} payload={row}>
        More
      </DropdownMenuTrigger>
      <NotificationMoreMenu
        handle={handle}
        handlers={handlers as unknown as NotificationRowHandlers}
      />
    </>,
  );
  fireEvent.click(screen.getByRole("button", { name: "More" }));

  return { handlers };
}

const items = () => screen.getAllByRole("menuitem").map((item) => item.textContent);

describe("the Later menu", () => {
  it("offers an hour from now, then the working-day times in the person's time zone", () => {
    openLaterMenu(notification());

    expect(items()).toEqual([
      expect.stringContaining("In 1 hour"),
      expect.stringContaining("Later today"),
      expect.stringContaining("Tomorrow morning"),
      expect.stringContaining("Monday"),
    ]);
  });

  it("sets the row aside for an hour", () => {
    const { onLater } = openLaterMenu(notification());

    fireEvent.click(screen.getByRole("menuitem", { name: /In 1 hour/ }));

    expect(onLater).toHaveBeenCalledWith("notif_1", NOW + 3_600);
  });

  it("sets the row aside until eight tomorrow morning in the person's time zone", () => {
    const { onLater } = openLaterMenu(notification());
    const tomorrow = snoozePresets(NOW, "America/Chicago").find((p) => p.key === "tomorrow");

    fireEvent.click(screen.getByRole("menuitem", { name: /Tomorrow morning/ }));

    // Saturday 08:00 in Chicago is 14:00 UTC.
    expect(tomorrow?.until).toBe(Date.UTC(2026, 10, 14, 14, 0, 0) / 1000);
    expect(onLater).toHaveBeenCalledWith("notif_1", tomorrow?.until);
  });
});

describe("the More menu", () => {
  it("marks an unread row read, and offers to turn its kind off and copy its link", () => {
    const row = notification();
    const { handlers } = openMoreMenu(row);

    expect(items()).toEqual(["Mark as read", "Turn off notifications like this", "Copy link"]);

    fireEvent.click(screen.getByRole("menuitem", { name: "Mark as read" }));
    expect(handlers.read).toHaveBeenCalledWith("notif_1");
  });

  it("marks a read row unread", () => {
    const { handlers } = openMoreMenu(notification({ readAt: NOW - 60 }));

    fireEvent.click(screen.getByRole("menuitem", { name: "Mark as unread" }));

    expect(handlers.unread).toHaveBeenCalledWith("notif_1");
  });

  it("turns the row's kind off", () => {
    const row = notification();
    const { handlers } = openMoreMenu(row);

    fireEvent.click(screen.getByRole("menuitem", { name: "Turn off notifications like this" }));

    expect(handlers.turnOff).toHaveBeenCalledWith(row);
  });

  it("copies the link the row opens", () => {
    const { handlers } = openMoreMenu(notification());

    fireEvent.click(screen.getByRole("menuitem", { name: "Copy link" }));

    expect(handlers.copyLink).toHaveBeenCalledWith("/dispatch/console?move=smv_1");
  });

  it("does not offer to turn off a kind the organization requires, or to copy a link it lacks", () => {
    openMoreMenu(notification({ eventType: "some.unregistered.event", data: null }));

    expect(items()).not.toContain("Turn off notifications like this");
    expect(items()).not.toContain("Copy link");
    expect(items()).toContain("Mark as read");
  });
});
