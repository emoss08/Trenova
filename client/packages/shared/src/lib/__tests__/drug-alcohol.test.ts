import { describe, expect, it } from "vitest";
import {
  clearinghouseResultTone,
  dotResultTone,
  drugAlcoholStatusMeta,
  isViolatingResult,
  projectedRoundTarget,
  violationProhibits,
} from "../drug-alcohol";

describe("isViolatingResult", () => {
  // An adulterated or substituted specimen is a refusal under 49 CFR 40.191,
  // so it has to read the same as a positive.
  it("counts refusals and tampering as violations", () => {
    for (const result of ["Positive", "Refusal", "Adulterated", "Substituted"]) {
      expect(isViolatingResult(result)).toBe(true);
    }
  });

  it("does not count a dilute negative", () => {
    expect(isViolatingResult("NegativeDilute")).toBe(false);
    expect(isViolatingResult("Invalid")).toBe(false);
  });
});

describe("violationProhibits", () => {
  // Follow-up testing happens after the driver is back at work. Reading it as a
  // bar would keep someone off the board for a year they are entitled to work.
  it("stops at the return-to-duty test", () => {
    expect(violationProhibits("Open")).toBe(true);
    expect(violationProhibits("SAPEvaluation")).toBe(true);
    expect(violationProhibits("RTDPending")).toBe(true);
    expect(violationProhibits("FollowUp")).toBe(false);
    expect(violationProhibits("Resolved")).toBe(false);
  });
});

describe("drugAlcoholStatusMeta", () => {
  it("says what each status means for dispatch", () => {
    expect(drugAlcoholStatusMeta("Prohibited").tone).toBe("inactive");
    expect(drugAlcoholStatusMeta("Clear").tone).toBe("active");
    expect(drugAlcoholStatusMeta("Pending").tone).toBe("warning");
  });

  // A status the client does not recognise must not read as clear.
  it("falls back to nothing-on-file", () => {
    expect(drugAlcoholStatusMeta("SomethingNew").label).toBe("Not on file");
  });
});

describe("tones", () => {
  it("grades results and query answers", () => {
    expect(dotResultTone("Positive")).toBe("inactive");
    expect(dotResultTone("NegativeDilute")).toBe("active");
    expect(dotResultTone("Pending")).toBe("warning");
    expect(clearinghouseResultTone("NoViolations")).toBe("active");
    expect(clearinghouseResultTone("ConsentDenied")).toBe("inactive");
    expect(clearinghouseResultTone("Pending")).toBe("warning");
  });
});

describe("projectedRoundTarget", () => {
  // 100 drivers at 50% a year over four rounds is 12.5, which rounds up:
  // taking 12 every quarter would finish the year at 48% and under the minimum.
  it("rounds a quarterly target up", () => {
    expect(projectedRoundTarget(100, 50, "Quarterly")).toBe(13);
    expect(projectedRoundTarget(100, 10, "Quarterly")).toBe(3);
    expect(projectedRoundTarget(100, 50, "Annual")).toBe(50);
  });

  it("never asks for more people than the pool holds", () => {
    expect(projectedRoundTarget(3, 100, "Monthly")).toBeLessThanOrEqual(3);
  });

  it("is zero when there is nobody or no rate", () => {
    expect(projectedRoundTarget(0, 50, "Quarterly")).toBe(0);
    expect(projectedRoundTarget(50, 0, "Quarterly")).toBe(0);
    expect(projectedRoundTarget(50, 50, "Never")).toBe(0);
  });
});
