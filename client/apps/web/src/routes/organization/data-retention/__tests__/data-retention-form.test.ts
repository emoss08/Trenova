import { dataRetentionSchema } from "@/types/data-retention";
import { describe, expect, it } from "vitest";
import {
  AI_AUDIT_RETENTION_DEFAULT_DAYS,
  AI_AUDIT_RETENTION_MIN_DAYS,
  AI_CORRECTION_RETENTION_DEFAULT_DAYS,
  AI_CORRECTION_RETENTION_MIN_DAYS,
  dataRetentionFormSchema,
  dataRetentionFormValues,
} from "../data-retention-form";

/*
The limits are the server's (services/tms/internal/core/domain/tenant/
dataretention.go): DefaultAIAuditRetentionDays 2555, seven years, and
MinAIAuditRetentionDays 365. The REST field is aiAuditRetentionPeriod.
*/

const valid = {
  auditRetentionPeriod: 120,
  ediInboundFileRetentionPeriod: 0,
  ediMessageRetentionPeriod: 0,
  aiFeedbackRetentionPeriod: 730,
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
      auditRetentionPeriod: 120,
      aiAuditRetentionPeriod: 3000,
    });
    expect(dataRetentionFormValues(saved).aiAuditRetentionPeriod).toBe(3000);

    const older = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditRetentionPeriod: 120,
    });
    expect(dataRetentionFormValues(older).aiAuditRetentionPeriod).toBe(2555);

    const zeroed = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditRetentionPeriod: 120,
      aiAuditRetentionPeriod: 0,
    });
    expect(dataRetentionFormValues(zeroed).aiAuditRetentionPeriod).toBe(2555);
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
      auditRetentionPeriod: 120,
      aiCorrectionRetentionPeriod: 0,
    });
    expect(dataRetentionFormValues(older).aiCorrectionRetentionPeriod).toBe(730);

    const saved = dataRetentionSchema.parse({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      auditRetentionPeriod: 120,
      aiCorrectionRetentionPeriod: 90,
    });
    expect(dataRetentionFormValues(saved).aiCorrectionRetentionPeriod).toBe(90);
  });
});
