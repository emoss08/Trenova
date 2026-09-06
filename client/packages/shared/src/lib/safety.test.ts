import { describe, expect, it } from "vitest";
import {
  describeSafetyEvent,
  describeSeverity,
  disciplinaryLevelMeta,
  nextDisciplinaryLevel,
  safetyRatingMeta,
  scoreRingValue,
  summariseInspections,
} from "./safety";

describe("safetyRatingMeta", () => {
  it("gives each rating a tone and orders them from best to worst", () => {
    const order = ["Excellent", "Good", "Watch", "AtRisk"] as const;
    const ranks = order.map((rating) => safetyRatingMeta(rating).rank);
    expect([...ranks].sort((a, b) => a - b)).toEqual(ranks);
    expect(safetyRatingMeta("Excellent").badgeVariant).toBe("active");
    expect(safetyRatingMeta("AtRisk").badgeVariant).toBe("inactive");
    expect(safetyRatingMeta("Watch").label).toBe("Watch");
  });
});

describe("scoreRingValue", () => {
  it("maps a 0-100 score onto the gauge and clamps out-of-range input", () => {
    expect(scoreRingValue(100)).toBe(1);
    expect(scoreRingValue(50)).toBe(0.5);
    expect(scoreRingValue(0)).toBe(0);
    expect(scoreRingValue(-10)).toBe(0);
    expect(scoreRingValue(140)).toBe(1);
  });
});

describe("disciplinaryLevelMeta and nextDisciplinaryLevel", () => {
  it("ranks the ladder and names the next rung", () => {
    expect(disciplinaryLevelMeta("Coaching").rank).toBe(1);
    expect(disciplinaryLevelMeta("Termination").rank).toBe(6);
    expect(disciplinaryLevelMeta("Termination").endsEmployment).toBe(true);
    expect(disciplinaryLevelMeta("FinalWarning").endsEmployment).toBe(false);

    expect(nextDisciplinaryLevel(null)).toBe("Coaching");
    expect(nextDisciplinaryLevel("Coaching")).toBe("VerbalWarning");
    expect(nextDisciplinaryLevel("Suspension")).toBe("Termination");
    expect(nextDisciplinaryLevel("Termination")).toBe("Termination");
  });
});

describe("describeSafetyEvent", () => {
  it("reads as a sentence for each kind, with the inspection outcome when there is one", () => {
    expect(describeSafetyEvent({ kind: "Accident", severity: "Major", preventable: true })).toBe(
      "Major accident, preventable",
    );
    expect(describeSafetyEvent({ kind: "Accident", severity: "Minor", preventable: false })).toBe(
      "Minor accident, non-preventable",
    );
    expect(describeSafetyEvent({ kind: "NearMiss", severity: "Minor" })).toBe("Near miss");
    expect(describeSafetyEvent({ kind: "Citation", severity: "Moderate" })).toBe(
      "Moderate citation",
    );
    expect(
      describeSafetyEvent({
        kind: "Inspection",
        severity: "Major",
        inspectionResult: "OutOfService",
        inspectionLevel: 1,
      }),
    ).toBe("Level 1 inspection — out of service");
    expect(
      describeSafetyEvent({ kind: "Inspection", severity: "Minor", inspectionResult: "Pass" }),
    ).toBe("Inspection — passed");
  });
});

describe("describeSeverity", () => {
  it("labels each severity", () => {
    expect(describeSeverity("Minor")).toBe("Minor");
    expect(describeSeverity("Critical")).toBe("Critical");
  });
});

describe("summariseInspections", () => {
  it("reports the clean rate as a percentage, or says there were none", () => {
    expect(summariseInspections({ inspections: 4, inspectionsPassed: 3 })).toBe(
      "3 of 4 clean (75%)",
    );
    expect(summariseInspections({ inspections: 1, inspectionsPassed: 1 })).toBe(
      "1 of 1 clean (100%)",
    );
    expect(summariseInspections({ inspections: 0, inspectionsPassed: 0 })).toBe(
      "No inspections in the last year",
    );
  });
});
