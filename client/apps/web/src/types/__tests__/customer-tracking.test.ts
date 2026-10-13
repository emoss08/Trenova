import { describe, expect, it } from "vitest";
import { customerTrackingSettingsSchema, splitRecipients } from "../customer-tracking";

const valid = {
  linkEnabled: true,
  visibleFields: ["Status", "Eta"],
  linkTtlDays: 30,
  notifyPickedUp: false,
  notifyDelayed: true,
  notifyDelivered: true,
  notifyPodReady: false,
  includeLink: true,
  recipients: "",
};

const issues = (value: Record<string, unknown>) => {
  const result = customerTrackingSettingsSchema.safeParse({ ...valid, ...value });
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
};

describe("splitRecipients", () => {
  it("splits, normalizes and deduplicates like the server", () => {
    expect(splitRecipients(" Ops@Example.com; ops@example.com,\nlane@example.com ")).toEqual([
      "ops@example.com",
      "lane@example.com",
    ]);
    expect(splitRecipients("  ")).toEqual([]);
  });
});

describe("customerTrackingSettingsSchema", () => {
  it("accepts the settings a customer starts with", () => {
    expect(issues({})).toEqual([]);
  });

  it("holds a link's life between one and 180 days", () => {
    expect(issues({ linkTtlDays: 0 })).toEqual(["linkTtlDays"]);
    expect(issues({ linkTtlDays: 181 })).toEqual(["linkTtlDays"]);
  });

  it("refuses an unknown field and a bad or overlong recipient list", () => {
    expect(issues({ visibleFields: ["Margin"] })).toEqual(["visibleFields.0"]);
    expect(issues({ recipients: "ops@example.com, not-an-email" })).toEqual(["recipients"]);
    const many = Array.from({ length: 21 }, (_, i) => `ops${i}@example.com`).join(", ");
    expect(issues({ recipients: many })).toEqual(["recipients"]);
  });
});
