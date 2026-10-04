import type { AgentControl } from "@/lib/graphql/agent-control";
import { describe, expect, it } from "vitest";
import { controlInput, promotionThresholdOptions } from "../agent-control-options";

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

/**
 * Every switch saves the whole control, so a change to one must carry the
 * others as they stand. Learning is stored as an off switch: turning learning
 * on sends learningOff false.
 */
describe("controlInput", () => {
  const current = {
    shadowMode: false,
    earnedAutonomy: true,
    promotionThreshold: 10,
    personMonthlyMessages: 200,
    learningOff: true,
    aiTrainingConsent: true,
  } as AgentControl;

  it("turns learning on or off without touching the other switches", () => {
    expect(controlInput(current, { learningOff: false })).toEqual({
      shadowMode: false,
      earnedAutonomy: true,
      promotionThreshold: 10,
      personMonthlyMessages: 200,
      learningOff: false,
    });
    expect(
      controlInput({ ...current, learningOff: false }, { learningOff: true }).learningOff,
    ).toBe(true);
  });

  it("keeps learning as it stands when another switch changes", () => {
    expect(controlInput(current, { shadowMode: true })).toMatchObject({
      shadowMode: true,
      learningOff: true,
    });
    expect(
      controlInput({ ...current, learningOff: false }, { promotionThreshold: 25 }),
    ).toMatchObject({ promotionThreshold: 25, learningOff: false });
  });

  it("sends training consent only when it is the switch being changed", () => {
    expect(controlInput(current, { learningOff: false })).not.toHaveProperty("aiTrainingConsent");
    expect(controlInput(current, { aiTrainingConsent: false })).toMatchObject({
      aiTrainingConsent: false,
      learningOff: true,
    });
  });
});
