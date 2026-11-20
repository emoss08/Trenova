import { describe, expect, it } from "vitest";
import {
  invoiceAdjustmentControlFormSchema,
  invoiceAdjustmentControlSchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { savedInvoiceAdjustmentControl } from "./invoice-adjustment-control-fixture";

const record = (changes: Partial<InvoiceAdjustmentControl>): InvoiceAdjustmentControl => ({
  ...savedInvoiceAdjustmentControl,
  ...changes,
});

function refusals(values: InvoiceAdjustmentControl) {
  const result = invoiceAdjustmentControlFormSchema.safeParse(values);
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
}

describe("invoice adjustment control contract", () => {
  it("reads the record the server sends, decimals kept as strings", () => {
    expect(invoiceAdjustmentControlSchema.parse(savedInvoiceAdjustmentControl)).toEqual(
      savedInvoiceAdjustmentControl,
    );
  });

  it("knows only the eligibility policies the server keeps", () => {
    expect(invoiceAdjustmentControlSchema.shape.paidInvoiceAdjustmentPolicy.options).toEqual([
      "Disallow",
      "AllowWithApproval",
    ]);
    expect(
      invoiceAdjustmentControlSchema.safeParse({
        ...savedInvoiceAdjustmentControl,
        paidInvoiceAdjustmentPolicy: "AllowWithoutApproval",
      }).success,
    ).toBe(false);
  });
});

describe("invoice adjustment control form", () => {
  it("accepts a new organization's record", () => {
    expect(refusals(savedInvoiceAdjustmentControl)).toEqual([]);
  });

  it("asks for approval amounts above zero while approval starts from an amount", () => {
    expect(
      refusals(record({ standardAdjustmentApprovalThreshold: "0", writeOffApprovalThreshold: "" })),
    ).toEqual([
      ["standardAdjustmentApprovalThreshold", "Enter an amount above zero"],
      ["writeOffApprovalThreshold", "Enter an amount above zero"],
    ]);
  });

  it("does not check an approval amount no policy uses", () => {
    expect(
      refusals(
        record({
          standardAdjustmentApprovalPolicy: "Always",
          standardAdjustmentApprovalThreshold: "",
          writeOffApprovalPolicy: "Disallow",
          writeOffApprovalThreshold: "0",
        }),
      ),
    ).toEqual([]);
  });

  it("asks for a tolerance the server can keep while changes in amount are reviewed", () => {
    for (const tolerance of ["", "-1", "1000"]) {
      expect(refusals(record({ rerateVarianceTolerancePercent: tolerance }))).toEqual([
        ["rerateVarianceTolerancePercent", "Enter a percentage from 0 to 999.99"],
      ]);
    }
    expect(refusals(record({ rerateVarianceTolerancePercent: "999.99" }))).toEqual([]);
  });
});
