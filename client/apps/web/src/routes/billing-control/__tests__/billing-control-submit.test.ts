import { describe, expect, it } from "vitest";
import type { BillingControl } from "@/types/billing-control";
import { prepareBillingControlForSubmit } from "../billing-control-submit";
import { billingControlValues } from "./billing-control-fixture";

// Drafts are made by hand in the fixture, which the save sends with no batch size.
const record = (changes: Partial<BillingControl>): BillingControl => ({
  ...billingControlValues,
  autoInvoiceBatchSize: null,
  ...changes,
});

describe("prepareBillingControlForSubmit", () => {
  it("sends hand-made drafts with no batch size and no draft notifications", () => {
    const sent = prepareBillingControlForSubmit(
      record({
        invoiceDraftCreationMode: "ManualOnly",
        autoInvoiceBatchSize: 50,
        notifyOnAutoInvoiceCreation: true,
      }),
    );

    expect(sent).toEqual(
      record({
        invoiceDraftCreationMode: "ManualOnly",
        autoInvoiceBatchSize: null,
        notifyOnAutoInvoiceCreation: false,
      }),
    );
  });

  it("keeps the batch size and notifications of drafts made on their own", () => {
    const values = record({
      invoiceDraftCreationMode: "AutomaticWhenTransferred",
      autoInvoiceBatchSize: 50,
      notifyOnAutoInvoiceCreation: true,
    });

    expect(prepareBillingControlForSubmit(values)).toEqual(values);
  });

  it.each(["Warn", "Block"] as const)(
    "sends skipping review as off when rate validation is %s, keeping the tolerance",
    (level) => {
      const sent = prepareBillingControlForSubmit(
        record({
          rateValidationEnforcement: level,
          rateVarianceAutoResolutionMode: "BypassReviewWithinTolerance",
          rateVarianceTolerancePercent: 2.5,
        }),
      );

      expect(sent).toEqual(
        record({
          rateValidationEnforcement: level,
          rateVarianceAutoResolutionMode: "Disabled",
          rateVarianceTolerancePercent: 2.5,
        }),
      );
    },
  );

  it("sends ignored rate validation with no tolerance and skipping review off", () => {
    const sent = prepareBillingControlForSubmit(
      record({
        rateValidationEnforcement: "Ignore",
        rateVarianceAutoResolutionMode: "BypassReviewWithinTolerance",
        rateVarianceTolerancePercent: 2.5,
      }),
    );

    expect(sent).toEqual(
      record({
        rateValidationEnforcement: "Ignore",
        rateVarianceAutoResolutionMode: "Disabled",
        rateVarianceTolerancePercent: 0,
      }),
    );
  });

  it("keeps skipping review within tolerance while rate validation requires review", () => {
    const values = record({
      rateValidationEnforcement: "RequireReview",
      rateVarianceAutoResolutionMode: "BypassReviewWithinTolerance",
      rateVarianceTolerancePercent: 2.5,
    });

    expect(prepareBillingControlForSubmit(values)).toEqual(values);
  });

  it("does not change the record it is given", () => {
    const values = record({ autoInvoiceBatchSize: 50 });
    prepareBillingControlForSubmit(values);

    expect(values.autoInvoiceBatchSize).toBe(50);
  });
});
