import { describe, expect, it } from "vitest";
import { ediWindowHasTraffic } from "../edi-summary";

const QUIET = {
  deliveryStatusCounts: [],
  ackStatusCounts: [],
  inboundFileStatusCounts: [],
  inboundTransferStatusCounts: [],
  overdueAckCount: 0,
  attentionItems: [],
};

describe("ediWindowHasTraffic", () => {
  it("is false when every count is absent or zero", () => {
    expect(ediWindowHasTraffic(QUIET)).toBe(false);
    expect(
      ediWindowHasTraffic({
        ...QUIET,
        deliveryStatusCounts: [{ status: "Sent", count: 0 }],
        ackStatusCounts: [{ status: "Pending", count: 0 }],
      }),
    ).toBe(false);
  });

  // A healthy window still has traffic: a sent delivery with nothing wrong
  // is the dashboard's good news, not an empty page.
  it("is true for a delivery in any state, including a good one", () => {
    expect(
      ediWindowHasTraffic({ ...QUIET, deliveryStatusCounts: [{ status: "Sent", count: 1 }] }),
    ).toBe(true);
  });

  it("is true for an acknowledgment, a file, a transfer, an overdue ack or a failure", () => {
    expect(
      ediWindowHasTraffic({ ...QUIET, ackStatusCounts: [{ status: "Accepted", count: 1 }] }),
    ).toBe(true);
    expect(
      ediWindowHasTraffic({
        ...QUIET,
        inboundFileStatusCounts: [{ status: "Processed", count: 2 }],
      }),
    ).toBe(true);
    expect(
      ediWindowHasTraffic({
        ...QUIET,
        inboundTransferStatusCounts: [{ status: "Approved", count: 1 }],
      }),
    ).toBe(true);
    expect(ediWindowHasTraffic({ ...QUIET, overdueAckCount: 1 })).toBe(true);
    expect(ediWindowHasTraffic({ ...QUIET, attentionItems: [{ id: "x" }] })).toBe(true);
  });
});
