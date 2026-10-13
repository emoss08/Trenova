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
