import { describe, expect, it } from "vitest";
import {
  caseClassificationLabel,
  classificationIsRecordable,
  classificationTone,
  formatRate,
  incidentRate,
  suggestClassification,
} from "../injury";

describe("classificationIsRecordable", () => {
  // First aid alone is explicitly not recordable (29 CFR 1904.7(b)(5)(ii)),
  // and that distinction is what the whole log turns on.
  it("excludes first aid", () => {
    expect(classificationIsRecordable("FirstAidOnly")).toBe(false);
    expect(classificationIsRecordable("NotRecordable")).toBe(false);
  });

  it("includes every column of the log", () => {
    for (const value of ["OtherRecordable", "JobTransferOrRestriction", "DaysAway", "Death"]) {
      expect(classificationIsRecordable(value)).toBe(true);
    }
  });
});

describe("suggestClassification", () => {
  // The log records the most serious outcome, so days away outrank a
  // restriction even when both happened.
  it("takes the most serious outcome", () => {
    expect(suggestClassification("MedicalTreatment", 3, 5)).toBe("DaysAway");
    expect(suggestClassification("FirstAid", 0, 5)).toBe("JobTransferOrRestriction");
    expect(suggestClassification("MedicalTreatment", 0, 0)).toBe("OtherRecordable");
    expect(suggestClassification("FirstAid", 0, 0)).toBe("FirstAidOnly");
    expect(suggestClassification("None", 0, 0)).toBe("NotRecordable");
  });
});

describe("incidentRate", () => {
  // 5 cases over 200,000 hours is exactly 5 per 100 full-time workers.
  it("is cases per 100 full-time workers", () => {
    expect(incidentRate(5, 200_000)).toBeCloseTo(5, 5);
    expect(incidentRate(1, 100_000)).toBeCloseTo(2, 5);
  });

  // A rate with no denominator is not a small number; it is not a number.
  it("is null without hours", () => {
    expect(incidentRate(5, 0)).toBeNull();
    expect(formatRate(incidentRate(5, 0))).toBe("—");
  });
});

describe("labels and tones", () => {
  it("uses the OSHA words", () => {
    expect(caseClassificationLabel("DaysAway")).toBe("Days away from work");
    expect(caseClassificationLabel("SomethingNew")).toBe("SomethingNew");
  });

  it("grades severity", () => {
    expect(classificationTone("Death")).toBe("inactive");
    expect(classificationTone("JobTransferOrRestriction")).toBe("warning");
    expect(classificationTone("FirstAidOnly")).toBe("secondary");
  });
});
