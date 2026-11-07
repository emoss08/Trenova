import { describe, expect, it } from "vitest";
import { statusLineState } from "../console-status-line";

describe("statusLineState", () => {
  it("is live only with a good read, a network and a connected stream", () => {
    expect(statusLineState({ isError: false }, true, "connected")).toBe("live");
  });

  it("says reconnecting while the stream that carries others' changes is down", () => {
    expect(statusLineState({ isError: false }, true, "connecting")).toBe("reconnecting");
    expect(statusLineState({ isError: false }, true, "disconnected")).toBe("reconnecting");
  });

  it("says offline without a network, whatever the stream last reported", () => {
    expect(statusLineState({ isError: false }, false, "connected")).toBe("offline");
  });

  it("puts a failed read first, since the board on screen is then out of date", () => {
    expect(statusLineState({ isError: true }, false, "disconnected")).toBe("failed");
  });
});
