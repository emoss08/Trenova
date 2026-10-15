import { describe, expect, it } from "vitest";
import { dispatchControlSchema } from "../dispatch-control";

const BASE = {
  enableAutoAssignment: false,
  autoAssignmentStrategy: "Proximity",
  enforceWorkerAssign: true,
  enforceTrailerContinuity: true,
  enforceHosCompliance: true,
  enforceWorkerPtaRestrictions: true,
  enforceWorkerTractorFleetContinuity: true,
  enforceDriverQualificationCompliance: true,
  enforceMedicalCertCompliance: true,
  enforceHazmatCompliance: true,
  enforceDrugAndAlcoholCompliance: true,
  complianceEnforcementLevel: "Warning",
  recordServiceFailures: "Never",
};

const exceptionIssues = (value: Record<string, unknown>) => {
  const result = dispatchControlSchema.safeParse({ ...BASE, ...value });
  if (result.success) {
    return [];
  }
  return result.error.issues
    .map((issue) => issue.path.join("."))
    .filter((path) => path.startsWith("exception"));
};

describe("dispatchControlSchema exception settings", () => {
  it("fills the server's defaults when a record predates the exception queue", () => {
    const parsed = dispatchControlSchema.parse(BASE);

    expect(parsed.exceptionDueCriticalMinutes).toBe(60);
    expect(parsed.exceptionDueWarningMinutes).toBe(240);
    expect(parsed.exceptionDueInfoMinutes).toBe(1440);
    expect(parsed.exceptionEscalationMinutes).toBe(60);
    expect(parsed.exceptionMaxEscalations).toBe(3);
    expect(parsed.exceptionMissedStopGraceMinutes).toBe(30);
    expect(parsed.exceptionPodGraceHours).toBe(24);
  });

  it("refuses values outside the server's ranges", () => {
    expect(
      exceptionIssues({
        exceptionDueCriticalMinutes: 4,
        exceptionDueInfoMinutes: 10081,
        exceptionEscalationMinutes: 1441,
        exceptionMaxEscalations: 11,
        exceptionMissedStopGraceMinutes: -1,
        exceptionPodGraceHours: 0,
      }).sort(),
    ).toEqual(
      [
        "exceptionDueCriticalMinutes",
        "exceptionDueInfoMinutes",
        "exceptionEscalationMinutes",
        "exceptionMaxEscalations",
        "exceptionMissedStopGraceMinutes",
        "exceptionPodGraceHours",
      ].sort(),
    );
  });

  it("accepts no default owner and no escalation contact", () => {
    expect(
      exceptionIssues({ exceptionDefaultAssigneeId: null, exceptionEscalationContactId: "" }),
    ).toEqual([]);
  });
});

const tenderIssues = (value: Record<string, unknown>) => {
  const result = dispatchControlSchema.safeParse({ ...BASE, ...value });
  if (result.success) {
    return [];
  }
  return result.error.issues
    .filter((issue) => issue.path.join(".").startsWith("tender"))
    .map((issue) => `${issue.path.join(".")}: ${issue.message}`);
};

describe("dispatchControlSchema tender settings", () => {
  it("fills the server's defaults when a record predates counter-offers", () => {
    const parsed = dispatchControlSchema.parse(BASE);

    expect(parsed.tenderCounterOffersEnabled).toBe(true);
    expect(parsed.tenderCounterMarginFloorPercent).toBe("10");
    expect(parsed.tenderAutoAcceptCounters).toBe(false);
    expect(parsed.tenderCounterDecisionMinutes).toBe(60);
    expect(parsed.tenderExhaustedAction).toBe("Notify");
    expect(parsed.tenderEscalationRateIncreasePercent).toBe("0");
    expect(parsed.tenderEscalationOfferTtlMinutes).toBe(60);
    expect(parsed.tenderMaxEscalations).toBe(1);
  });

  it("reads a percentage the server sent as a number back as the same digits", () => {
    const parsed = dispatchControlSchema.parse({
      ...BASE,
      tenderCounterMarginFloorPercent: 7.125,
      tenderEscalationRateIncreasePercent: "12.5",
    });

    expect(parsed.tenderCounterMarginFloorPercent).toBe("7.125");
    expect(parsed.tenderEscalationRateIncreasePercent).toBe("12.5");
  });

  it("allows a negative margin floor down to -100%", () => {
    expect(tenderIssues({ tenderCounterMarginFloorPercent: "-100" })).toEqual([]);
    expect(tenderIssues({ tenderCounterMarginFloorPercent: "-100.5" })).toEqual([
      "tenderCounterMarginFloorPercent: Margin floor cannot be below -100%",
    ]);
  });

  it("holds the rate increase between 0 and 100% and to four decimals", () => {
    expect(tenderIssues({ tenderEscalationRateIncreasePercent: "-1" })).toEqual([
      "tenderEscalationRateIncreasePercent: Rate increase cannot be below 0%",
    ]);
    expect(tenderIssues({ tenderEscalationRateIncreasePercent: "100.01" })).toEqual([
      "tenderEscalationRateIncreasePercent: Rate increase cannot exceed 100%",
    ]);
    expect(tenderIssues({ tenderEscalationRateIncreasePercent: "1.23456" })).toEqual([
      "tenderEscalationRateIncreasePercent: Rate increase must be a number with at most 4 decimals",
    ]);
  });

  it("holds the counter window, escalated offer expiry and escalation limit to the server's bounds", () => {
    expect(
      tenderIssues({
        tenderCounterDecisionMinutes: 4,
        tenderEscalationOfferTtlMinutes: 10081,
        tenderMaxEscalations: 6,
      }),
    ).toEqual([
      "tenderCounterDecisionMinutes: Counter decision time must be at least 5 minutes",
      "tenderEscalationOfferTtlMinutes: Escalated offers cannot stay open beyond 10080 minutes",
      "tenderMaxEscalations: Escalations cannot exceed 5",
    ]);
  });

  it("refuses an exhausted-tender action the server does not know", () => {
    expect(tenderIssues({ tenderExhaustedAction: "Retry" })).toHaveLength(1);
  });
});
