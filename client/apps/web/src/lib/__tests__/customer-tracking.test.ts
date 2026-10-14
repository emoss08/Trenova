import type {
  CustomerStatusNotificationRow,
  CustomerTrackingLinkRow,
} from "@/lib/graphql/customer-tracking";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  TRACKING_FIELDS,
  linkState,
  liveLinkCount,
  skipReasonLabel,
  statusEmailSteps,
  statusEventLabel,
  trackingFieldLabel,
} from "../customer-tracking";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);
const NOW = 1_800_000_000;

function link(overrides: Partial<CustomerTrackingLinkRow> = {}): CustomerTrackingLinkRow {
  return {
    id: "cstl_1",
    shipmentId: "shp_1",
    customerId: "cus_1",
    purpose: "Shared",
    recipientEmail: null,
    expiresAt: NOW + 3600,
    revokedAt: null,
    lastViewedAt: null,
    viewCount: 0,
    createdById: null,
    createdAt: NOW - 60,
    ...overrides,
  } as CustomerTrackingLinkRow;
}

describe("linkState", () => {
  it("is revoked before it is expired", () => {
    expect(linkState(link({ revokedAt: NOW - 1, expiresAt: NOW - 10 }), NOW)).toBe("revoked");
    expect(linkState(link({ expiresAt: NOW }), NOW)).toBe("expired");
    expect(linkState(link(), NOW)).toBe("live");
  });

  it("counts only the links that still open", () => {
    expect(
      liveLinkCount(
        [link(), link({ id: "b", revokedAt: NOW }), link({ id: "c", expiresAt: 1 })],
        NOW,
      ),
    ).toBe(1);
  });
});

describe("labels", () => {
  it("names every event, status and skip reason", () => {
    expect(statusEventLabel(t, "PodReady")).toBe("Proof of delivery ready");
    expect(skipReasonLabel(t, "NoRecipients")).toBe("Nobody to send it to");
    expect(skipReasonLabel(t, null)).toBeNull();
  });

  it("labels every field a customer can be shown", () => {
    const labels = TRACKING_FIELDS.map((field) => trackingFieldLabel(t, field));
    expect(new Set(labels).size).toBe(TRACKING_FIELDS.length);
    expect(TRACKING_FIELDS).toHaveLength(13);
  });
});

function notification(
  overrides: Partial<CustomerStatusNotificationRow> = {},
): CustomerStatusNotificationRow {
  return {
    id: "csn_1",
    shipmentId: "shp_1",
    event: "PickedUp",
    stopId: "stp_1",
    occurredAt: NOW,
    estimatedAt: null,
    status: "Sent",
    skipReason: null,
    recipients: ["shipping@customer.com"],
    attempts: 1,
    lastError: null,
    sentAt: NOW + 30,
    createdAt: NOW,
    ...overrides,
  } as CustomerStatusNotificationRow;
}

describe("statusEmailSteps", () => {
  const rules = { PickedUp: true, Delayed: true, Delivered: true, PodReady: false };

  it("lists the four emails in the order a shipment earns them", () => {
    expect(statusEmailSteps(rules, []).map((step) => step.event)).toEqual([
      "PickedUp",
      "Delayed",
      "Delivered",
      "PodReady",
    ]);
  });

  it("pairs each email with whether the rules send it and the latest one owed", () => {
    const steps = statusEmailSteps(rules, [
      notification({ id: "a", event: "Delayed", occurredAt: NOW, status: "Sent" }),
      notification({ id: "b", event: "Delayed", occurredAt: NOW + 600, status: "Pending" }),
    ]);
    const delayed = steps.find((step) => step.event === "Delayed");
    expect(delayed?.enabled).toBe(true);
    expect(delayed?.count).toBe(2);
    expect(delayed?.latest?.id).toBe("b");
    expect(steps.find((step) => step.event === "PodReady")).toMatchObject({
      enabled: false,
      latest: null,
      count: 0,
    });
  });

  it("still shows an email sent before its rule was turned off", () => {
    const steps = statusEmailSteps({ ...rules, PickedUp: false }, [notification()]);
    expect(steps[0]).toMatchObject({ enabled: false, count: 1 });
    expect(steps[0].latest?.status).toBe("Sent");
  });
});
