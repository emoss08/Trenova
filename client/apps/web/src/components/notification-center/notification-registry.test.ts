import type { Notification } from "@trenova/shared/types/notification";
import { describe, expect, it } from "vitest";
import { getNotificationDescriptor, getNotificationLink } from "./notification-registry";

const DISPATCH_EVENT_TYPES = [
  "tender_accepted",
  "tender_needs_review",
  "tender_waterfall_exhausted",
  "rate_confirmation_issue_failed",
  "tender_delivery_failed",
  "tender_entries_skipped",
] as const;

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "not_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    targetUserId: null,
    eventType: "tender_accepted",
    priority: "high",
    channel: "global",
    title: "Tender accepted",
    message: "Blue Ridge Freight accepted the tender",
    data: null,
    relatedEntities: null,
    source: "tender",
    readAt: null,
    dismissedAt: null,
    createdAt: 1_700_000_000,
    ...overrides,
  };
}

describe("notification registry — dispatch tender descriptors", () => {
  const fallback = getNotificationDescriptor("some.unregistered.event");

  it.each(DISPATCH_EVENT_TYPES)("registers %s under the Dispatch category", (eventType) => {
    const descriptor = getNotificationDescriptor(eventType);

    expect(descriptor).not.toBe(fallback);
    expect(descriptor.category).toBe("Dispatch");
    expect(descriptor.icon).not.toBe(fallback.icon);
  });

  it.each(DISPATCH_EVENT_TYPES)("prefers the producer's data.link for %s", (eventType) => {
    const link = getNotificationLink(
      notification({ eventType, data: { link: "/dispatch/console?move=smv_01" } }),
    );

    expect(link).toBe("/dispatch/console?move=smv_01");
  });

  it.each(DISPATCH_EVENT_TYPES)(
    "falls back to the dispatch console for %s when data carries no link",
    (eventType) => {
      expect(getNotificationLink(notification({ eventType }))).toBe("/dispatch/console");
    },
  );

  it.each(DISPATCH_EVENT_TYPES)(
    "falls back to the dispatch console for %s when data.link is not a usable string",
    (eventType) => {
      expect(getNotificationLink(notification({ eventType, data: { link: "" } }))).toBe(
        "/dispatch/console",
      );
      expect(getNotificationLink(notification({ eventType, data: { link: 42 } }))).toBe(
        "/dispatch/console",
      );
    },
  );
});

describe("notification registry — generic link handling", () => {
  it("surfaces data.link for an unregistered event type", () => {
    const link = getNotificationLink(
      notification({ eventType: "some.unregistered.event", data: { link: "/dispatch/console" } }),
    );

    expect(link).toBe("/dispatch/console");
  });

  it("surfaces data.link for a tca.* data alert", () => {
    const link = getNotificationLink(
      notification({ eventType: "tca.table_change", data: { link: "/shipment-management" } }),
    );

    expect(link).toBe("/shipment-management");
  });

  it("still returns null for a link-less notification", () => {
    expect(getNotificationLink(notification({ eventType: "some.unregistered.event" }))).toBeNull();
    expect(getNotificationLink(notification({ eventType: "tca.table_change" }))).toBeNull();
  });
});
