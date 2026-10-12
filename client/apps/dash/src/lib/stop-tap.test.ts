import { describe, expect, it } from "vitest";
import { stopTap } from "./stop-tap";

describe("stopTap", () => {
  it("stamps the tap with the device time in unix seconds", () => {
    const tap = stopTap("stp_1", "Arrive", new Date("2026-01-07T14:15:30.900Z"));

    expect(tap).toMatchObject({ stopId: "stp_1", action: "Arrive", occurredAt: 1767795330 });
  });

  it("gives every tap its own id", () => {
    const first = stopTap("stp_1", "Depart");
    const second = stopTap("stp_1", "Depart");

    expect(first.clientEventId).not.toBe("");
    expect(first.clientEventId).not.toBe(second.clientEventId);
  });
});
