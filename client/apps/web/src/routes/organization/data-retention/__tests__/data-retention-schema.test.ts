import { dataRetentionSchema } from "@/types/data-retention";
import { describe, expect, it } from "vitest";
import {
  DRIVER_QUALIFICATION_RETENTION_DEFAULT_DAYS,
  DRIVER_QUALIFICATION_RETENTION_MIN_DAYS,
  AI_AUDIT_RETENTION_DEFAULT_DAYS,
  AI_AUDIT_RETENTION_MIN_DAYS,
  AI_CORRECTION_RETENTION_DEFAULT_DAYS,
  AI_CORRECTION_RETENTION_MIN_DAYS,
  dataRetentionFormSchema,
  dataRetentionFormValues,
  retentionCutoff,
} from "../data-retention-form";

/*
The limits are the server's (services/tms/internal/modules/identity/domain/
dataretention/dataretention.go): DefaultAIAuditRetentionDays 2555, seven years, and
MinAIAuditRetentionDays 365; the audit floors 90, 730, 2555 and 730 with a ceiling
of 3650; 0 on the EDI and evaluation-case periods keeps everything.
*/

const valid = {
  auditAccessRetentionDays: 90,
  auditOperationalRetentionDays: 730,
  auditFinancialRetentionDays: 2555,
  auditSecurityRetentionDays: 730,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 0,
  aiFeedbackRetentionPeriod: 730,
  driverQualificationRetentionPeriod: 1095,
  agentEvalCaseRetentionPeriod: 365,
  aiAuditRetentionPeriod: 2555,
  aiCorrectionRetentionPeriod: 730,
};

describe("AI audit trail retention", () => {
  it("keeps seven years by default and a year at the least", () => {
    expect(AI_AUDIT_RETENTION_DEFAULT_DAYS).toBe(2555);
    expect(AI_AUDIT_RETENTION_MIN_DAYS).toBe(365);
  });

  it("accepts a year or more", () => {
    expect(
      dataRetentionFormSchema.safeParse({ ...valid, aiAuditRetentionPeriod: 365 }).success,
    ).toBe(true);
    expect(
      dataRetentionFormSchema.safeParse({ ...valid, aiAuditRetentionPeriod: 3650 }).success,
    ).toBe(true);
  });

  it("refuses less than a year, a fraction of a day, and nothing at all", () => {
    for (const value of [364, 0, -1, 400.5]) {
      const parsed = dataRetentionFormSchema.safeParse({ ...valid, aiAuditRetentionPeriod: value });
      expect(parsed.success, String(value)).toBe(false);
      expect(parsed.error?.issues[0]?.path).toEqual(["aiAuditRetentionPeriod"]);
    }
    const { aiAuditRetentionPeriod: _omitted, ...missing } = valid;
    expect(dataRetentionFormSchema.safeParse(missing).success).toBe(false);
  });

  // An organization saved before the field existed reads back without it:
  // the page offers the server's default rather than a zero it would refuse.
  it("reads a saved setting, and the default when none was saved", () => {
    const saved = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditAccessRetentionDays: 90,
      auditOperationalRetentionDays: 730,
      auditFinancialRetentionDays: 2555,
      auditSecurityRetentionDays: 730,
      aiAuditRetentionPeriod: 3000,
    });
    expect(dataRetentionFormValues(saved).aiAuditRetentionPeriod).toBe(3000);

    const older = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditAccessRetentionDays: 90,
      auditOperationalRetentionDays: 730,
      auditFinancialRetentionDays: 2555,
      auditSecurityRetentionDays: 730,
    });
    expect(dataRetentionFormValues(older).aiAuditRetentionPeriod).toBe(2555);

    const zeroed = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditAccessRetentionDays: 90,
      auditOperationalRetentionDays: 730,
      auditFinancialRetentionDays: 2555,
      auditSecurityRetentionDays: 730,
      aiAuditRetentionPeriod: 0,
    });
    expect(dataRetentionFormValues(zeroed).aiAuditRetentionPeriod).toBe(2555);
  });
});

describe("driver qualification retention", () => {
  it("keeps three years by default and as the least", () => {
    expect(DRIVER_QUALIFICATION_RETENTION_DEFAULT_DAYS).toBe(1095);
    expect(DRIVER_QUALIFICATION_RETENTION_MIN_DAYS).toBe(1095);
  });

  it("refuses less than three years with the floor's own message", () => {
    for (const value of [1094, 0, -1, 1100.5]) {
      const parsed = dataRetentionFormSchema.safeParse({
        ...valid,
        driverQualificationRetentionPeriod: value,
      });
      expect(parsed.success, String(value)).toBe(false);
      expect(parsed.error?.issues[0]?.path).toEqual(["driverQualificationRetentionPeriod"]);
    }
    const short = dataRetentionFormSchema.safeParse({
      ...valid,
      driverQualificationRetentionPeriod: 1094,
    });
    expect(short.error?.issues[0]?.message).toBe(
      "Driver qualification files must be kept for at least 1,095 days (three years) after employment ends",
    );
    expect(
      dataRetentionFormSchema.safeParse({ ...valid, driverQualificationRetentionPeriod: 1095 })
        .success,
    ).toBe(true);
  });

  it("offers the default when an older setting has none", () => {
    const older = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      driverQualificationRetentionPeriod: 0,
    });
    expect(dataRetentionFormValues(older).driverQualificationRetentionPeriod).toBe(1095);
  });
});

describe("AI correction retention", () => {
  it("keeps two years by default and a month at the least", () => {
    expect(AI_CORRECTION_RETENTION_DEFAULT_DAYS).toBe(730);
    expect(AI_CORRECTION_RETENTION_MIN_DAYS).toBe(30);
  });

  it("refuses less than a month and a fraction of a day", () => {
    for (const value of [29, 0, -1, 45.5]) {
      const parsed = dataRetentionFormSchema.safeParse({
        ...valid,
        aiCorrectionRetentionPeriod: value,
      });
      expect(parsed.success, String(value)).toBe(false);
      expect(parsed.error?.issues[0]?.path).toEqual(["aiCorrectionRetentionPeriod"]);
    }
    expect(
      dataRetentionFormSchema.safeParse({ ...valid, aiCorrectionRetentionPeriod: 30 }).success,
    ).toBe(true);
  });

  it("offers the default when an older setting has none", () => {
    const older = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditAccessRetentionDays: 90,
      auditOperationalRetentionDays: 730,
      auditFinancialRetentionDays: 2555,
      auditSecurityRetentionDays: 730,
      aiCorrectionRetentionPeriod: 0,
    });
    expect(dataRetentionFormValues(older).aiCorrectionRetentionPeriod).toBe(730);

    const saved = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditAccessRetentionDays: 90,
      auditOperationalRetentionDays: 730,
      auditFinancialRetentionDays: 2555,
      auditSecurityRetentionDays: 730,
      aiCorrectionRetentionPeriod: 90,
    });
    expect(dataRetentionFormValues(saved).aiCorrectionRetentionPeriod).toBe(90);
  });
});

describe("retentionCutoff", () => {
  const now = 1_793_000_000;

  it("reaches back exactly the window's days from now", () => {
    expect(retentionCutoff(1, now)).toBe(now - 86_400);
    expect(retentionCutoff(120, now)).toBe(now - 120 * 86_400);
    expect(retentionCutoff(2555, now)).toBe(now - 2555 * 86_400);
  });

  it("has no cutoff for a window that keeps everything", () => {
    expect(retentionCutoff(0, now)).toBeNull();
    expect(retentionCutoff(Number.NaN, now)).toBeNull();
  });
});

describe("every period", () => {
  it("accepts the server's floors and ceilings, and 0 where 0 keeps everything", () => {
    expect(dataRetentionFormSchema.safeParse(valid).success).toBe(true);
    expect(
      dataRetentionFormSchema.safeParse({
        ...valid,
        auditAccessRetentionDays: 3650,
        auditOperationalRetentionDays: 3650,
        auditFinancialRetentionDays: 3650,
        auditSecurityRetentionDays: 3650,
        agentEvalCaseRetentionPeriod: 0,
      }).success,
    ).toBe(true);
  });

  it("refuses an audit period under its floor or over 3,650 days with the server's words", () => {
    const cases = [
      ["auditAccessRetentionDays", 89, "Reads and downloads are kept for at least 90 days"],
      ["auditOperationalRetentionDays", 729, "Operational changes are kept for at least 730 days"],
      [
        "auditFinancialRetentionDays",
        2554,
        "Financial changes are kept for at least 2,555 days (seven years)",
      ],
      ["auditSecurityRetentionDays", 729, "Security entries are kept for at least 730 days"],
      ["auditSecurityRetentionDays", 3651, "Audit entries are kept for at most 3,650 days"],
    ] as const;
    for (const [field, value, message] of cases) {
      const parsed = dataRetentionFormSchema.safeParse({ ...valid, [field]: value });
      expect(parsed.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
        [[field], message],
      ]);
    }
  });

  it("refuses a negative period where 0 keeps everything", () => {
    for (const field of [
      "ediInboundFileRetentionPeriod",
      "ediMessageRetentionPeriod",
      "agentEvalCaseRetentionPeriod",
    ]) {
      const parsed = dataRetentionFormSchema.safeParse({ ...valid, [field]: -1 });
      expect(parsed.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
        [[field], "A period cannot be negative"],
      ]);
    }
  });

  // An emptied number field holds null; the refusal says what to enter, not a type error.
  it("asks for a number of days when a period is emptied", () => {
    const parsed = dataRetentionFormSchema.safeParse({
      ...valid,
      ediMessageRetentionPeriod: null,
      aiFeedbackRetentionPeriod: null,
    });
    expect(parsed.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
      [["ediMessageRetentionPeriod"], "Enter how many days to keep message payloads"],
      [["aiFeedbackRetentionPeriod"], "Enter how many days to keep AI feedback"],
    ]);
  });

  it("refuses a fraction of a day", () => {
    const parsed = dataRetentionFormSchema.safeParse({ ...valid, ediMessageRetentionPeriod: 1.5 });
    expect(parsed.error?.issues.map((issue) => issue.message)).toEqual([
      "Enter a whole number of days",
    ]);
  });
});
