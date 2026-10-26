import { describe, expect, it } from "vitest";
import { ediReadinessLabel } from "../service-failure-edi-readiness";

describe("ediReadinessLabel", () => {
  it("supports old and new preview ready skipped reasons", () => {
    expect(ediReadinessLabel("skipped", "ready")).toBe("Ready for generation");
    expect(ediReadinessLabel("skipped", "ready_for_generation")).toBe("Ready for generation");
  });

  it("names why the partner's Lading version sends no 214", () => {
    expect(
      ediReadinessLabel("skipped", "the partner's flow sends no 214 on a service failure"),
    ).toBe("Not in the partner's flow");
    expect(ediReadinessLabel("skipped", "the partner has no active Lading version")).toBe(
      "No active version",
    );
    expect(
      ediReadinessLabel("blocked", "the partner's 214 map or rules refuse this service failure"),
    ).toBe("Blocked");
  });

  it("reads a 214 the flow holds for its next batch as queued", () => {
    expect(ediReadinessLabel("queued")).toBe("Queued");
    expect(ediReadinessLabel("skipped", "the partner's flow sends it with its next batch")).toBe(
      "Queued",
    );
  });
});
