import { customerBillingProfileSchema } from "@trenova/shared/types/customer";
import { describe, expect, it } from "vitest";
import { invoiceAdjustmentControlSchema } from "@/types/invoice-adjustment-control";
import {
  invoiceAdjustmentPreviewSchema,
  invoiceAdjustmentSchema,
} from "@/types/invoice-adjustment";

describe("supporting document requirement removal", () => {
  it("keeps no attachment requirement on the invoice adjustment control", () => {
    expect(invoiceAdjustmentControlSchema.shape).not.toHaveProperty(
      "adjustmentAttachmentRequirement",
    );
  });

  it("keeps no supporting document policy on the customer billing profile", () => {
    const keys = Object.keys(customerBillingProfileSchema.safeParse({}).data ?? {});
    expect(keys).not.toContain("invoiceAdjustmentSupportingDocumentPolicy");
  });

  it("drops the policy fields from adjustments and previews", () => {
    for (const schema of [invoiceAdjustmentSchema, invoiceAdjustmentPreviewSchema]) {
      expect(schema.shape).not.toHaveProperty("supportingDocumentsRequired");
      expect(schema.shape).not.toHaveProperty("customerSupportingDocumentPolicy");
      expect(schema.shape).not.toHaveProperty("supportingDocumentPolicySource");
    }
  });
});
