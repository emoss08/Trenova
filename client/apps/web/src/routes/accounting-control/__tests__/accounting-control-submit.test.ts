import { describe, expect, it } from "vitest";
import type { AccountingControl } from "@/types/accounting-control";
import { sanitizeAccountingControlForSubmit } from "../accounting-control-submit";
import { savedAccountingControl } from "./accounting-control-fixture";

const record = (changes: Partial<AccountingControl>): AccountingControl => ({
  ...savedAccountingControl,
  ...changes,
});

describe("sanitizeAccountingControlForSubmit", () => {
  it("sends a single-currency record with overrides disallowed and no FX accounts", () => {
    const sent = sanitizeAccountingControlForSubmit(
      record({
        currencyMode: "SingleCurrency",
        exchangeRateOverridePolicy: "RequireApproval",
        realizedFxGainAccountId: "gla_gain",
        realizedFxLossAccountId: "gla_loss",
      }),
    );

    expect(sent).toEqual(
      record({
        currencyMode: "SingleCurrency",
        exchangeRateOverridePolicy: "Disallow",
        realizedFxGainAccountId: null,
        realizedFxLossAccountId: null,
      }),
    );
  });

  it("keeps a multi-currency record's override policy and FX accounts", () => {
    const values = record({
      currencyMode: "MultiCurrency",
      exchangeRateOverridePolicy: "Allow",
      realizedFxGainAccountId: "gla_gain",
      realizedFxLossAccountId: "gla_loss",
    });

    expect(sanitizeAccountingControlForSubmit(values)).toEqual(values);
  });

  it("sends no auto-post events when journals are posted manually", () => {
    const sent = sanitizeAccountingControlForSubmit(
      record({
        journalPostingMode: "Manual",
        autoPostSourceEvents: ["InvoicePosted", "VendorBillPosted"],
      }),
    );

    expect(sent).toEqual(record({ journalPostingMode: "Manual", autoPostSourceEvents: [] }));
  });

  it("keeps the auto-post events when journals post automatically", () => {
    const values = record({
      journalPostingMode: "Automatic",
      autoPostSourceEvents: [
        "InvoicePosted",
        "CreditMemoPosted",
        "DebitMemoPosted",
        "VendorBillPosted",
      ],
    });

    expect(sanitizeAccountingControlForSubmit(values)).toEqual(values);
  });

  it("turns manual journal approval off when manual journals are disallowed", () => {
    const sent = sanitizeAccountingControlForSubmit(
      record({ manualJournalEntryPolicy: "Disallow", requireManualJeApproval: true }),
    );

    expect(sent).toEqual(
      record({ manualJournalEntryPolicy: "Disallow", requireManualJeApproval: false }),
    );
  });

  it("keeps manual journal approval for adjustment-only manual journals", () => {
    const values = record({
      manualJournalEntryPolicy: "AdjustmentOnly",
      requireManualJeApproval: true,
    });

    expect(sanitizeAccountingControlForSubmit(values)).toEqual(values);
  });

  it("turns period close approval off when a scheduled job closes periods", () => {
    const sent = sanitizeAccountingControlForSubmit(
      record({ periodCloseMode: "SystemScheduled", requirePeriodCloseApproval: true }),
    );

    expect(sent).toEqual(
      record({ periodCloseMode: "SystemScheduled", requirePeriodCloseApproval: false }),
    );
  });

  it("zeroes the tolerance and drops the close gate when reconciliation is off", () => {
    const sent = sanitizeAccountingControlForSubmit(
      record({
        reconciliationMode: "Disabled",
        reconciliationToleranceAmount: 25.5,
        requireReconciliationToClose: true,
        notifyOnReconciliationException: true,
      }),
    );

    expect(sent).toEqual(
      record({
        reconciliationMode: "Disabled",
        reconciliationToleranceAmount: 0,
        requireReconciliationToClose: false,
        notifyOnReconciliationException: true,
      }),
    );
  });

  it("keeps the tolerance and close gate while reconciliation blocks posting", () => {
    const values = record({
      reconciliationMode: "BlockPosting",
      reconciliationToleranceAmount: 25.5,
      requireReconciliationToClose: true,
    });

    expect(sanitizeAccountingControlForSubmit(values)).toEqual(values);
  });

  it("leaves the values it was given untouched", () => {
    const values = record({
      journalPostingMode: "Manual",
      autoPostSourceEvents: ["InvoicePosted"],
    });

    sanitizeAccountingControlForSubmit(values);

    expect(values.autoPostSourceEvents).toEqual(["InvoicePosted"]);
  });
});
