import { describe, expect, it } from "vitest";
import { DEFAULT_DISPATCH_ECONOMICS, dispatchControlSchema } from "../dispatch-control";

const BASE = {
  enableAutoAssignment: false,
  autoProposeCoverage: false,
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

describe("dispatchControlSchema economics", () => {
  it("gives a record that predates costs and prices the server's defaults", () => {
    const parsed = dispatchControlSchema.parse(BASE);

    expect(parsed.economics).toEqual(DEFAULT_DISPATCH_ECONOMICS);
    expect(parsed.learnDispatchStyle).toBe(false);
    expect(parsed.planForEndMarket).toBe(false);
    expect(parsed.anticipateFreight).toBe(false);
    expect(parsed.economics.anticipationMaxCents).toBe(40_000);
  });

  it("keeps anticipating freight and the most a likely load is worth as the server sent them", () => {
    const parsed = dispatchControlSchema.parse({
      ...BASE,
      anticipateFreight: true,
      economics: { ...DEFAULT_DISPATCH_ECONOMICS, anticipationMaxCents: 125_000 },
    });

    expect(parsed.anticipateFreight).toBe(true);
    expect(parsed.economics.anticipationMaxCents).toBe(125_000);
    expect(dispatchControlSchema.parse(parsed)).toEqual(parsed);
  });

  it("gives economics saved without a likely-load cap the server's default cap", () => {
    const withoutCap: Record<string, unknown> = { ...DEFAULT_DISPATCH_ECONOMICS };
    delete withoutCap.anticipationMaxCents;
    const parsed = dispatchControlSchema.parse({
      ...BASE,
      economics: { ...withoutCap, emptyMileCents: 250 },
    });

    expect(parsed.economics.anticipationMaxCents).toBe(40_000);
    expect(parsed.economics.emptyMileCents).toBe(250);
  });

  it("holds the most a likely load is worth between $0 and $2,000 in whole cents", () => {
    const capIssues = (anticipationMaxCents: number) => {
      const result = dispatchControlSchema.safeParse({
        ...BASE,
        economics: { ...DEFAULT_DISPATCH_ECONOMICS, anticipationMaxCents },
      });
      return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
    };

    expect(capIssues(0)).toEqual([]);
    expect(capIssues(200_000)).toEqual([]);
    expect(capIssues(200_001)).toEqual(["economics.anticipationMaxCents"]);
    expect(capIssues(-1)).toEqual(["economics.anticipationMaxCents"]);
    expect(capIssues(100.5)).toEqual(["economics.anticipationMaxCents"]);
  });

  it("refuses a negative cost on the field that holds it", () => {
    const result = dispatchControlSchema.safeParse({
      ...BASE,
      economics: { ...DEFAULT_DISPATCH_ECONOMICS, emptyMileCents: -1 },
    });

    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.path.join("."))).toContain(
      "economics.emptyMileCents",
    );
  });
});
