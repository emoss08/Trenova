import { describe, expect, it } from "vitest";
import { ediReadinessLabel } from "../service-failure-edi-readiness";

describe("ediReadinessLabel", () => {
  it("supports old and new preview ready skipped reasons", () => {
    expect(ediReadinessLabel("skipped", "ready")).toBe("Ready for generation");
    expect(ediReadinessLabel("skipped", "ready_for_generation")).toBe("Ready for generation");
  });
});
