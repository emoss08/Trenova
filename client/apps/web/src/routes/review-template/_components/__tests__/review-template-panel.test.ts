import type { ReviewTemplateRow } from "@/lib/graphql/performance-review";
import { reviewTemplateFormSchema } from "@trenova/shared/types/performance-review";
import { describe, expect, it } from "vitest";
import { buildReviewTemplateDefaults, toReviewTemplateInput } from "../review-template-panel";

const row: ReviewTemplateRow = {
  id: "prt_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  code: "DRIVER-ANNUAL",
  name: "Annual Driver Review",
  description: "Yearly review",
  status: "Active",
  isDefault: true,
  cadenceMonths: 12,
  items: [
    { key: "safety", label: "Safe driving", description: "Inspections", weight: 3 },
    { key: "service", label: "Customer service", description: null, weight: 2 },
  ],
  openReviewCount: 2,
  version: 3,
  createdAt: 1,
  updatedAt: 2,
};

describe("review template form mapping", () => {
  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildReviewTemplateDefaults(row);
    expect(defaults.cadenceMonths).toBe(12);
    expect(defaults.items).toHaveLength(2);
    expect(defaults.items[1].description).toBeNull();

    const parsed = reviewTemplateFormSchema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    const input = toReviewTemplateInput(defaults, row.version);
    expect(input).toMatchObject({ code: "DRIVER-ANNUAL", cadenceMonths: 12, version: 3 });
    expect(input.items[0]).toMatchObject({ key: "safety", weight: 3 });
    expect(input.items[1].description).toBeUndefined();
  });

  it("uppercases the code and starts new templates with one weighted item", () => {
    const defaults = buildReviewTemplateDefaults(null);
    expect(defaults.items).toHaveLength(1);
    expect(defaults.items[0].weight).toBeGreaterThan(0);
    defaults.code = "otr-quarterly";
    defaults.name = "OTR Quarterly";
    expect(toReviewTemplateInput(defaults).code).toBe("OTR-QUARTERLY");
  });

  it("rejects duplicate item keys, a default that is not active, and a bad key", () => {
    const base = { ...buildReviewTemplateDefaults(row), code: "STD", name: "Standard" };

    const duplicate = reviewTemplateFormSchema.safeParse({
      ...base,
      items: [base.items[0], { ...base.items[0] }],
    });
    expect(duplicate.success).toBe(false);
    expect(duplicate.error?.issues.map((issue) => issue.path.join("."))).toContain("items.1.key");

    const inactiveDefault = reviewTemplateFormSchema.safeParse({
      ...base,
      status: "Inactive",
      isDefault: true,
    });
    expect(inactiveDefault.success).toBe(false);

    const badKey = reviewTemplateFormSchema.safeParse({
      ...base,
      items: [{ ...base.items[0], key: "Safe Driving" }],
    });
    expect(badKey.success).toBe(false);

    const noItems = reviewTemplateFormSchema.safeParse({ ...base, items: [] });
    expect(noItems.success).toBe(false);
  });
});
