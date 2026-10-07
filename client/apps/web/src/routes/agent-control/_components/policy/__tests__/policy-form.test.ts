import { describe, expect, it } from "vitest";
import { promotesOnSave, toControlInput, type PolicyFormValues } from "../policy-form";

const loaded: PolicyFormValues = {
  earnedAutonomy: false,
  promotionThreshold: 10,
  learnsFromWork: true,
  personMonthlyMessages: 0,
  aiTrainingConsent: false,
  version: 4,
};

describe("toControlInput", () => {
  it("keeps the pause switch and leaves consent out unless it changed", () => {
    const input = toControlInput({ ...loaded, learnsFromWork: false }, loaded, true);

    expect(input).toEqual({
      shadowMode: true,
      earnedAutonomy: false,
      promotionThreshold: 10,
      learningOff: true,
      personMonthlyMessages: 0,
      version: 4,
    });
  });

  it("sends consent when it changed", () => {
    expect(toControlInput({ ...loaded, aiTrainingConsent: true }, loaded, false)).toMatchObject({
      aiTrainingConsent: true,
    });
  });
});

describe("promotesOnSave", () => {
  it("is true when earned autonomy turns on or its threshold drops", () => {
    expect(promotesOnSave({ ...loaded, earnedAutonomy: true }, loaded)).toBe(true);
    const on = { ...loaded, earnedAutonomy: true };
    expect(promotesOnSave({ ...on, promotionThreshold: 5 }, on)).toBe(true);
    expect(promotesOnSave({ ...on, promotionThreshold: 25 }, on)).toBe(false);
    expect(promotesOnSave(loaded, loaded)).toBe(false);
  });
});
