import { describe, expect, it } from "vitest";
import {
  accountingControlFormSchema,
  accountingControlSchema,
  type AccountingControl,
} from "@/types/accounting-control";
import { sanitizeAccountingControlForSubmit } from "../accounting-control-submit";
import { savedAccountingControl } from "./accounting-control-fixture";

const record = (changes: Partial<AccountingControl>): AccountingControl => ({
  ...savedAccountingControl,
  ...changes,
});

function refusals(values: AccountingControl) {
  const result = accountingControlFormSchema.safeParse(values);
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
}

const coreAccounts = {
  defaultRevenueAccountId: "gla_revenue",
  defaultExpenseAccountId: "gla_expense",
  defaultArAccountId: "gla_ar",
  defaultApAccountId: "gla_ap",
} satisfies Partial<AccountingControl>;

const accrualAutomatic = record({
  journalPostingMode: "Automatic",
  autoPostSourceEvents: [
    "InvoicePosted",
    "CreditMemoPosted",
    "DebitMemoPosted",
    "VendorBillPosted",
  ],
  ...coreAccounts,
});

const cashAutomatic = record({
  accountingBasis: "Cash",
  revenueRecognitionPolicy: "OnCashReceipt",
  expenseRecognitionPolicy: "OnCashDisbursement",
  journalPostingMode: "Automatic",
  autoPostSourceEvents: ["CustomerPaymentPosted", "VendorPaymentPosted"],
  ...coreAccounts,
  defaultCashAccountId: "gla_cash",
  defaultUnappliedCashAccountId: "gla_unapplied",
});

// From the server: createAccountingBasisRule, createJournalPostingRule, createCurrencyRule
// and createReconciliationRule in accountingcontrolservice/validator.go.
describe("accounting control contract", () => {
  it("reads every record the server sends, even one its validator would refuse", () => {
    const refused = record({
      accountingBasis: "Cash",
      journalPostingMode: "Automatic",
      reconciliationMode: "WarnOnly",
      currencyMode: "MultiCurrency",
    });
    expect(accountingControlSchema.safeParse(refused).success).toBe(true);
  });
});

describe("accounting control form", () => {
  it("accepts a new organization's record", () => {
    expect(refusals(savedAccountingControl)).toEqual([]);
  });

  it("accepts records the server accepts: automatic posting on either basis, and every check on", () => {
    expect(refusals(accrualAutomatic)).toEqual([]);
    expect(refusals(cashAutomatic)).toEqual([]);
    expect(
      refusals(
        record({
          ...accrualAutomatic,
          currencyMode: "MultiCurrency",
          exchangeRateOverridePolicy: "RequireApproval",
          realizedFxGainAccountId: "gla_fx_gain",
          realizedFxLossAccountId: "gla_fx_loss",
          reconciliationMode: "BlockPosting",
          reconciliationToleranceAmount: 0.5,
          requireReconciliationToClose: true,
        }),
      ),
    ).toEqual([]);
  });

  it("refuses recognition policies the basis does not allow", () => {
    expect(refusals(record({ accountingBasis: "Cash" }))).toEqual([
      ["revenueRecognitionPolicy", "Choose the revenue recognition this accounting basis allows"],
      ["expenseRecognitionPolicy", "Choose the expense recognition this accounting basis allows"],
    ]);
  });

  it("refuses automatic posting with no events and no accounts", () => {
    expect(refusals(record({ journalPostingMode: "Automatic" }))).toEqual([
      ["autoPostSourceEvents", "Choose at least one event that posts journals"],
      ["defaultRevenueAccountId", "Choose a revenue account"],
      ["defaultExpenseAccountId", "Choose an expense account"],
      ["defaultArAccountId", "Choose an accounts receivable account"],
      ["defaultApAccountId", "Choose an accounts payable account"],
    ]);
  });

  it("refuses automatic posting that leaves out an event the policies need", () => {
    expect(
      refusals(record({ ...accrualAutomatic, autoPostSourceEvents: ["InvoicePosted"] })),
    ).toEqual([
      ["autoPostSourceEvents", "Include every event your recognition policies depend on"],
    ]);
  });

  it("refuses invoice and memo events while revenue is recognized on cash receipt", () => {
    expect(
      refusals(
        record({
          ...cashAutomatic,
          autoPostSourceEvents: ["InvoicePosted", "CustomerPaymentPosted", "VendorPaymentPosted"],
        }),
      ),
    ).toEqual([
      [
        "autoPostSourceEvents",
        "Invoices and memos cannot post journals while revenue is recognized on cash receipt",
      ],
    ]);
  });

  it("requires cash and unapplied cash once customer payments post", () => {
    const withoutCash = { defaultCashAccountId: null, defaultUnappliedCashAccountId: null };
    const expected = [
      ["defaultCashAccountId", "Choose a cash account"],
      ["defaultUnappliedCashAccountId", "Choose an unapplied cash account"],
    ];

    expect(refusals(record({ ...cashAutomatic, ...withoutCash }))).toEqual(expected);
    expect(
      refusals(
        record({
          ...accrualAutomatic,
          autoPostSourceEvents: [...accrualAutomatic.autoPostSourceEvents, "CustomerPaymentPosted"],
        }),
      ),
    ).toEqual(expected);
  });

  it("requires both realized FX accounts under several currencies", () => {
    expect(refusals(record({ currencyMode: "MultiCurrency" }))).toEqual([
      ["realizedFxGainAccountId", "Choose a realized FX gain account"],
      ["realizedFxLossAccountId", "Choose a realized FX loss account"],
    ]);
  });

  it("requires a tolerance above zero while reconciliation runs", () => {
    for (const reconciliationMode of ["WarnOnly", "BlockPosting"] as const) {
      expect(refusals(record({ reconciliationMode, reconciliationToleranceAmount: 0 }))).toEqual([
        ["reconciliationToleranceAmount", "Enter a tolerance above zero"],
      ]);
    }
  });

  // These are rows the page hides; prepare sends them as the server requires, and the
  // resolver runs before prepare, so the form must not refuse them.
  it("leaves the hidden rows prepare fixes to prepare", () => {
    const hidden = record({
      autoPostSourceEvents: ["InvoicePosted"],
      manualJournalEntryPolicy: "Disallow",
      requireManualJeApproval: true,
      periodCloseMode: "SystemScheduled",
      requirePeriodCloseApproval: true,
      reconciliationMode: "Disabled",
      reconciliationToleranceAmount: 5,
      requireReconciliationToClose: true,
      currencyMode: "SingleCurrency",
      exchangeRateOverridePolicy: "Allow",
      realizedFxGainAccountId: "gla_fx_gain",
      realizedFxLossAccountId: "gla_fx_loss",
    });

    expect(refusals(hidden)).toEqual([]);
    expect(refusals(sanitizeAccountingControlForSubmit(hidden))).toEqual([]);
  });
});
