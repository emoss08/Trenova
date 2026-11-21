import { describe, expect, it } from "vitest";
import { dataRetentionFormSchema } from "../data-retention-form";
import { dataRetentionTabs } from "../data-retention-tabs";
import type { UpdateDataRetentionRequest } from "@/types/data-retention";

/*
The fields PUT /data-retention/ takes (dataretentionservice.UpdateDataRetentionRequest). The
form edits exactly these; the record's ids, version and timestamps are not sent.
*/
const REQUEST_FIELDS: readonly (keyof UpdateDataRetentionRequest)[] = [
  "auditAccessRetentionDays",
  "auditOperationalRetentionDays",
  "auditFinancialRetentionDays",
  "auditSecurityRetentionDays",
  "ediInboundFileRetentionPeriod",
  "ediMessageRetentionPeriod",
  "driverQualificationRetentionPeriod",
  "agentEvalCaseRetentionPeriod",
  "aiFeedbackRetentionPeriod",
  "aiAuditRetentionPeriod",
  "aiCorrectionRetentionPeriod",
];

describe("data retention tabs", () => {
  it("puts every period the save sends on exactly one tab", () => {
    const placed = dataRetentionTabs().flatMap((tab) => tab.fields);

    expect([...placed].sort()).toEqual([...REQUEST_FIELDS].sort());
    expect(Object.keys(dataRetentionFormSchema.shape).sort()).toEqual([...REQUEST_FIELDS].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("has one tab, retention, for every organization", () => {
    expect(dataRetentionTabs().map((tab) => tab.value)).toEqual(["retention"]);
  });
});
