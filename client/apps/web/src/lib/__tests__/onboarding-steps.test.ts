import {
  firstInvalidOnboardingStep,
  nextOnboardingStep,
  ONBOARDING_STEPS,
  onboardingProgress,
  onboardingStepIndex,
  onboardingTimezoneTiles,
  REVIEW_STEP_INDEX,
  SAMPLE_DATA_RECORD_COUNT,
} from "@/lib/onboarding-form";
import { describe, expect, it } from "vitest";

describe("nextOnboardingStep", () => {
  it("moves one turn forward while answering for the first time", () => {
    expect(nextOnboardingStep(0, 0)).toBe(1);
    expect(nextOnboardingStep(3, 3)).toBe(4);
  });

  it("returns straight to the furthest turn after an edit", () => {
    expect(nextOnboardingStep(0, REVIEW_STEP_INDEX)).toBe(REVIEW_STEP_INDEX);
    expect(nextOnboardingStep(2, 5)).toBe(5);
  });

  it("never runs past the review", () => {
    expect(nextOnboardingStep(REVIEW_STEP_INDEX, REVIEW_STEP_INDEX)).toBe(REVIEW_STEP_INDEX);
  });
});

describe("onboardingProgress", () => {
  it("starts at 4% and reaches 78% at the review", () => {
    expect(onboardingProgress(0, "setup")).toBe(4);
    expect(onboardingProgress(REVIEW_STEP_INDEX, "setup")).toBe(78);
  });

  it("holds at 92% while building or after a failure and fills on ready", () => {
    expect(onboardingProgress(REVIEW_STEP_INDEX, "building")).toBe(92);
    expect(onboardingProgress(REVIEW_STEP_INDEX, "failed")).toBe(92);
    expect(onboardingProgress(REVIEW_STEP_INDEX, "ready")).toBe(100);
  });
});

describe("firstInvalidOnboardingStep", () => {
  it("finds the earliest turn that owns an invalid field", () => {
    const invalid = new Set(["organization.dotNumber", "operationType"]);
    expect(firstInvalidOnboardingStep((field) => invalid.has(field))).toBe(
      onboardingStepIndex("ids"),
    );
  });

  it("puts every address field on the address turn", () => {
    for (const field of [
      "organization.addressLine1",
      "organization.city",
      "organization.stateId",
      "organization.postalCode",
    ]) {
      expect(firstInvalidOnboardingStep((candidate) => candidate === field)).toBe(
        onboardingStepIndex("address"),
      );
    }
  });

  it("answers -1 when nothing a turn asks for is invalid", () => {
    expect(firstInvalidOnboardingStep(() => false)).toBe(-1);
  });

  it("asks every form field exactly once", () => {
    const fields = ONBOARDING_STEPS.flatMap((step) => step.fields);
    expect(new Set(fields).size).toBe(fields.length);
    expect(fields).toHaveLength(10);
  });
});

describe("onboardingTimezoneTiles", () => {
  it("leads with the browser's US zone without repeating it", () => {
    expect(onboardingTimezoneTiles("America/Denver")).toEqual([
      "America/Denver",
      "America/New_York",
      "America/Chicago",
      "America/Phoenix",
      "America/Los_Angeles",
      "America/Anchorage",
      "Pacific/Honolulu",
    ]);
  });

  it("leads with a known zone outside the US too", () => {
    const tiles = onboardingTimezoneTiles("Europe/London");
    expect(tiles[0]).toBe("Europe/London");
    expect(tiles).toHaveLength(8);
  });

  it("lists only the US zones when the browser's zone is unknown", () => {
    expect(onboardingTimezoneTiles("")).toHaveLength(7);
    expect(onboardingTimezoneTiles("Mars/Olympus_Mons")[0]).toBe("America/New_York");
  });
});

it("counts the sample records the server creates", () => {
  expect(SAMPLE_DATA_RECORD_COUNT).toBe(11);
});
