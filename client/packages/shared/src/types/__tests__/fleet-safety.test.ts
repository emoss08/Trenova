import { describe, expect, it } from "vitest";
import { dashControlFormSchema } from "../driver-pay";
import { safetyViolationFormSchema } from "../fleet-safety";

function baseViolation() {
  return {
    basic: "HOSCompliance" as const,
    code: "395.8",
    description: "No record of duty status",
    severityWeight: 5,
    outOfService: false,
  };
}

describe("safetyViolationFormSchema", () => {
  it("accepts a violation off an inspection report", () => {
    expect(safetyViolationFormSchema.safeParse(baseViolation()).success).toBe(true);
  });

  // The FMCSA weights run 1 to 10 and the published tables are what a clerk
  // keys from. A weight nobody could look up is not evidence of anything.
  it("refuses a weight outside the published scale", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: 0 }).success,
    ).toBe(false);
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: 11 }).success,
    ).toBe(false);
  });

  // The field is bound to a numeric control. A schema that accepted the string
  // a select would have written would let the two disagree silently.
  it("refuses a weight that arrived as text", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), severityWeight: "5" }).success,
    ).toBe(false);
  });

  it("refuses a BASIC outside the seven", () => {
    expect(
      safetyViolationFormSchema.safeParse({ ...baseViolation(), basic: "Paperwork" }).success,
    ).toBe(false);
  });
});

function baseDashControl() {
  return {
    requireLoadAcknowledgment: true,
    allowLoadRefusals: true,
    allowStopActions: true,
    allowLoadDocumentUpload: true,
    allowLoadComments: true,
    showLoadPay: true,
    showPayEstimates: true,
    allowExpenseSubmission: true,
    requireExpenseReceipt: false,
    allowSettlementDisputes: true,
    allowProfileDocumentUpload: true,
    allowContactInfoEdit: true,
    requireContactChangeApproval: false,
    allowPtoRequests: true,
    sendCredentialReminders: true,
    driverDigestCadence: "Immediate" as const,
    driverDigestWeekday: "1" as const,
    enableDetentionAlerts: true,
    detentionAlertThresholdMinutes: 120,
  };
}

describe("dashControlFormSchema digest settings", () => {
  // The weekday is bound to a select, which writes strings. The schema has to
  // accept what the control actually produces; the integer the column wants is
  // made at the call site.
  it("takes the weekday as the string a select writes", () => {
    expect(dashControlFormSchema.safeParse(baseDashControl()).success).toBe(true);
    expect(
      dashControlFormSchema.safeParse({ ...baseDashControl(), driverDigestWeekday: 1 }).success,
    ).toBe(false);
  });

  it("refuses a day that is not a day of the week", () => {
    expect(
      dashControlFormSchema.safeParse({ ...baseDashControl(), driverDigestWeekday: "7" }).success,
    ).toBe(false);
  });

  // A digest with reminders switched off would collect obligations and send
  // nothing, which reads as a working setting that quietly does nothing.
  it("refuses a round-up with driver reminders switched off", () => {
    const result = dashControlFormSchema.safeParse({
      ...baseDashControl(),
      driverDigestCadence: "Weekly",
      sendCredentialReminders: false,
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.path).toEqual(["driverDigestCadence"]);
    }
  });

  it("allows reminders off while every driver is told one at a time", () => {
    expect(
      dashControlFormSchema.safeParse({
        ...baseDashControl(),
        sendCredentialReminders: false,
      }).success,
    ).toBe(true);
  });
});
