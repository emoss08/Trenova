import { describe, expect, it } from "vitest";
import { promotionThresholdOptions } from "../agent-control-options";

/**
 * The threshold is chosen from a short list, but a value set another way
 * (the API, an older build) must still appear as the selected option rather
 * than leaving the control with nothing pressed.
 */
describe("promotionThresholdOptions", () => {
  it("offers the presets in order", () => {
    expect(promotionThresholdOptions(10).map((option) => option.value)).toEqual([5, 10, 25, 50]);
  });

  it("keeps an off-preset value in the list, in order", () => {
    expect(promotionThresholdOptions(15).map((option) => option.value)).toEqual([
      5, 10, 15, 25, 50,
    ]);
  });

  it("labels each option by its count", () => {
    expect(promotionThresholdOptions(10).find((option) => option.value === 25)?.label).toBe("25");
  });
});
