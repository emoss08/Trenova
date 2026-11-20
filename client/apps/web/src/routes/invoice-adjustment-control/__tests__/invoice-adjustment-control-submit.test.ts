import { describe, expect, it } from "vitest";
import type { InvoiceAdjustmentControl } from "@/types/invoice-adjustment-control";
import { prepareInvoiceAdjustmentControl } from "../invoice-adjustment-control-submit";
import { savedInvoiceAdjustmentControl } from "./invoice-adjustment-control-fixture";

const record = (changes: Partial<InvoiceAdjustmentControl>): InvoiceAdjustmentControl => ({
  ...savedInvoiceAdjustmentControl,
  ...changes,
});

describe("prepareInvoiceAdjustmentControl", () => {
  it("sends a new organization's record as it is", () => {
    expect(prepareInvoiceAdjustmentControl(savedInvoiceAdjustmentControl)).toEqual(
      savedInvoiceAdjustmentControl,
    );
  });

  // The server refuses RequireReopen unless adjustments book on the invoice's date.
  it("stops requiring a reopen once adjustments always book in the next open period", () => {
    const sent = prepareInvoiceAdjustmentControl(
      record({
        adjustmentAccountingDatePolicy: "AlwaysNextOpen",
        closedPeriodAdjustmentPolicy: "RequireReopen",
      }),
    );

    expect(sent).toEqual(
      record({
        adjustmentAccountingDatePolicy: "AlwaysNextOpen",
        closedPeriodAdjustmentPolicy: "Disallow",
      }),
    );
  });

  // The server refuses next-period posting with approval while nothing needs approval.
  it("drops next-period posting with approval when it is hidden and nothing needs approval", () => {
    const sent = prepareInvoiceAdjustmentControl(
      record({
        adjustmentAccountingDatePolicy: "AlwaysNextOpen",
        closedPeriodAdjustmentPolicy: "PostInNextOpenPeriodWithApproval",
        standardAdjustmentApprovalPolicy: "None",
      }),
    );

    expect(sent).toEqual(
      record({
        adjustmentAccountingDatePolicy: "AlwaysNextOpen",
        closedPeriodAdjustmentPolicy: "Disallow",
        standardAdjustmentApprovalPolicy: "None",
      }),
    );
  });

  it("keeps a hidden closed-period policy the server accepts", () => {
    const values = record({
      adjustmentAccountingDatePolicy: "AlwaysNextOpen",
      closedPeriodAdjustmentPolicy: "PostInNextOpenPeriodWithApproval",
      standardAdjustmentApprovalPolicy: "Always",
    });

    expect(prepareInvoiceAdjustmentControl(values)).toEqual(values);
  });

  it("keeps a closed-period policy the person can see", () => {
    const values = record({ closedPeriodAdjustmentPolicy: "RequireReopen" });

    expect(prepareInvoiceAdjustmentControl(values)).toEqual(values);
  });

  // The server refuses any over-credit but Block while unapplied credit is not allowed.
  it("blocks over-crediting when unapplied credit is not allowed", () => {
    const sent = prepareInvoiceAdjustmentControl(
      record({ customerCreditBalancePolicy: "Disallow", overCreditPolicy: "AllowWithApproval" }),
    );

    expect(sent).toEqual(
      record({ customerCreditBalancePolicy: "Disallow", overCreditPolicy: "Block" }),
    );
  });

  it("keeps over-crediting with approval while unapplied credit is allowed", () => {
    const values = record({
      customerCreditBalancePolicy: "AllowUnappliedCredit",
      overCreditPolicy: "AllowWithApproval",
    });

    expect(prepareInvoiceAdjustmentControl(values)).toEqual(values);
  });

  it("sends a hidden approval amount that was emptied as zero", () => {
    const sent = prepareInvoiceAdjustmentControl(
      record({
        standardAdjustmentApprovalPolicy: "Always",
        standardAdjustmentApprovalThreshold: "",
        writeOffApprovalPolicy: "Disallow",
        writeOffApprovalThreshold: " ",
      }),
    );

    expect(sent).toEqual(
      record({
        standardAdjustmentApprovalPolicy: "Always",
        standardAdjustmentApprovalThreshold: "0",
        writeOffApprovalPolicy: "Disallow",
        writeOffApprovalThreshold: "0",
      }),
    );
  });

  it("keeps a hidden approval amount that was typed", () => {
    const values = record({
      standardAdjustmentApprovalPolicy: "None",
      standardAdjustmentApprovalThreshold: "250.00",
    });

    expect(prepareInvoiceAdjustmentControl(values)).toEqual(values);
  });

  // The server refuses a negative tolerance whatever the review policy.
  it("sends a hidden rerate tolerance the server would refuse as zero", () => {
    for (const tolerance of ["", "-1", "1000"]) {
      expect(
        prepareInvoiceAdjustmentControl(
          record({
            replacementInvoiceReviewPolicy: "AlwaysRequireReview",
            rerateVarianceTolerancePercent: tolerance,
          }),
        ),
      ).toEqual(
        record({
          replacementInvoiceReviewPolicy: "AlwaysRequireReview",
          rerateVarianceTolerancePercent: "0",
        }),
      );
    }
  });

  it("keeps a hidden rerate tolerance the server accepts", () => {
    const values = record({
      replacementInvoiceReviewPolicy: "NoAdditionalReview",
      rerateVarianceTolerancePercent: "2.5",
    });

    expect(prepareInvoiceAdjustmentControl(values)).toEqual(values);
  });
});
