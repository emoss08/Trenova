import { describe, expect, it } from "vitest";
import { billingControlSchema, type BillingControl } from "@/types/billing-control";
import { billingControlValues, savedBillingControl } from "./billing-control-fixture";

const refusals = (changes: Partial<BillingControl>) => {
  const result = billingControlSchema.safeParse({ ...billingControlValues, ...changes });
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
};

describe("billingControlSchema", () => {
  it("reads the server's record, with an empty schedule as none and decimals as numbers", () => {
    expect(billingControlSchema.parse(savedBillingControl)).toEqual(billingControlValues);
  });

  it("reads a record whose reminder days are null as the default schedule", () => {
    expect(
      billingControlSchema.parse({ ...savedBillingControl, paymentReminderDays: null })
        .paymentReminderDays,
    ).toEqual([1, 15, 30]);
  });

  it("accepts the closed record as it stands", () => {
    expect(refusals({})).toEqual([]);
  });

  it("asks for a schedule and a batch size when shipments move to the queue on their own", () => {
    expect(
      refusals({
        billingQueueTransferMode: "AutomaticWhenReady",
        billingQueueTransferSchedule: null,
        billingQueueTransferBatchSize: 0,
      }),
    ).toEqual([
      ["billingQueueTransferSchedule", "Choose how often shipments move to the queue"],
      ["billingQueueTransferBatchSize", "Move at least one shipment per batch"],
    ]);
  });

  it("asks for a batch size when drafts are created on their own", () => {
    expect(
      refusals({
        invoiceDraftCreationMode: "AutomaticWhenTransferred",
        autoInvoiceBatchSize: null,
      }),
    ).toEqual([["autoInvoiceBatchSize", "Create at least one draft per batch"]]);
  });

  it("refuses posting on its own while drafts are made by hand", () => {
    expect(
      refusals({
        invoiceDraftCreationMode: "ManualOnly",
        invoicePostingMode: "AutomaticWhenNoBlockingExceptions",
      }),
    ).toEqual([
      [
        "invoicePostingMode",
        "Invoices can post on their own only when drafts are created on their own",
      ],
    ]);
  });

  it("asks for a tolerance above zero to skip review within it", () => {
    expect(
      refusals({
        rateValidationEnforcement: "RequireReview",
        rateVarianceAutoResolutionMode: "BypassReviewWithinTolerance",
        rateVarianceTolerancePercent: 0,
      }),
    ).toEqual([
      ["rateVarianceTolerancePercent", "Set a tolerance above zero to skip review within it"],
    ]);
  });

  // The page hides skipping review under any other level, and the save sends it as off.
  it("leaves a hidden skip-review setting to the save under another level", () => {
    expect(
      refusals({
        rateValidationEnforcement: "Warn",
        rateVarianceAutoResolutionMode: "BypassReviewWithinTolerance",
        rateVarianceTolerancePercent: 0,
      }),
    ).toEqual([]);
  });

  it.each([
    [[], "Choose at least one day past due to send a reminder on"],
    [[1, 2, 3, 4, 5, 6, 7], "Send at most six reminders per invoice"],
    [[0, 15], "Reminder days must be between 1 and 365 days past due"],
    [[1, 366], "Reminder days must be between 1 and 365 days past due"],
    [[15, 15], "List reminder days from earliest to latest, each once"],
  ])("refuses the reminder days %j while reminders are on", (days, message) => {
    expect(refusals({ paymentRemindersEnabled: true, paymentReminderDays: days })).toEqual([
      ["paymentReminderDays", message],
    ]);
  });

  it("ignores the reminder days while reminders are off", () => {
    expect(refusals({ paymentRemindersEnabled: false, paymentReminderDays: [] })).toEqual([]);
  });

  it("refuses a negative minimum late charge", () => {
    expect(refusals({ lateChargeMinimumAmount: -1 })).toEqual([
      ["lateChargeMinimumAmount", "The minimum late charge cannot be negative"],
    ]);
  });
});
