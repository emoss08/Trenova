import type { CustomerTrackingLinkRow } from "@/lib/graphql/customer-tracking";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  TRACKING_FIELDS,
  linkState,
  liveLinkCount,
  notificationStatusAttrs,
  skipReasonLabel,
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
    expect(notificationStatusAttrs(t).Failed.phase).toBe("failed");
    expect(notificationStatusAttrs(t).Sent.phase).toBe("complete");
    expect(skipReasonLabel(t, "NoRecipients")).toBe("Nobody to send it to");
    expect(skipReasonLabel(t, null)).toBeNull();
  });

  it("labels every field a customer can be shown", () => {
    const labels = TRACKING_FIELDS.map((field) => trackingFieldLabel(t, field));
    expect(new Set(labels).size).toBe(TRACKING_FIELDS.length);
    expect(TRACKING_FIELDS).toHaveLength(13);
  });
});
