import { describe, expect, it } from "vitest";
import {
  customerTrackingProfileSchema,
  defaultCustomerTrackingProfile,
  splitRecipients,
} from "../customer-tracking";

const issues = (value: Record<string, unknown>) => {
  const result = customerTrackingProfileSchema.safeParse({
    ...defaultCustomerTrackingProfile(),
    ...value,
  });
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

describe("customerTrackingProfileSchema", () => {
  it("accepts the settings a customer starts with", () => {
    expect(issues({})).toEqual([]);
  });

  it("accepts the null id and start time the server sends for settings never saved", () => {
    expect(issues({ id: null, notificationsEnabledAt: null })).toEqual([]);
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
