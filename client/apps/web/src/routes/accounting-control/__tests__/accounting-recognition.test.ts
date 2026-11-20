import { describe, expect, it } from "vitest";
import {
  blockedSourceEvents,
  RECOGNITION_BY_BASIS,
  requiredSourceEvents,
  sourceEventsForRecognition,
} from "../accounting-recognition";

// From the server: ValidRevenueRecognitionPolicies / ValidExpenseRecognitionPolicies in
// enums.go, and the events createJournalPostingRule in validator.go requires and blocks.
describe("accounting recognition", () => {
  it("allows accrual books one policy each: invoices and vendor bills", () => {
    expect(RECOGNITION_BY_BASIS.Accrual).toEqual({
      revenue: "OnInvoicePost",
      expense: "OnVendorBillPost",
    });
  });

  it("allows cash books one policy each: cash received and cash paid", () => {
    expect(RECOGNITION_BY_BASIS.Cash).toEqual({
      revenue: "OnCashReceipt",
      expense: "OnCashDisbursement",
    });
  });

  it("requires invoices, memos and vendor bills to post automatically on accrual books", () => {
    expect(requiredSourceEvents(RECOGNITION_BY_BASIS.Accrual)).toEqual([
      "InvoicePosted",
      "CreditMemoPosted",
      "DebitMemoPosted",
      "VendorBillPosted",
    ]);
  });

  it("requires customer and vendor payments to post automatically on cash books", () => {
    expect(requiredSourceEvents(RECOGNITION_BY_BASIS.Cash)).toEqual([
      "CustomerPaymentPosted",
      "VendorPaymentPosted",
    ]);
  });

  it("blocks invoices and memos from posting when revenue is recognized on cash receipt", () => {
    expect(blockedSourceEvents("OnCashReceipt")).toEqual([
      "InvoicePosted",
      "CreditMemoPosted",
      "DebitMemoPosted",
    ]);
    expect(blockedSourceEvents("OnInvoicePost")).toEqual([]);
  });

  it("swaps accrual's required events for cash's and keeps every other choice", () => {
    expect(
      sourceEventsForRecognition(
        [
          "InvoicePosted",
          "CreditMemoPosted",
          "DebitMemoPosted",
          "VendorBillPosted",
          "FactoringFunded",
        ],
        RECOGNITION_BY_BASIS.Accrual,
        RECOGNITION_BY_BASIS.Cash,
      ),
    ).toEqual(["CustomerPaymentPosted", "VendorPaymentPosted", "FactoringFunded"]);
  });

  it("swaps cash's required events for accrual's, in the server's order", () => {
    expect(
      sourceEventsForRecognition(
        ["CustomerPaymentPosted", "VendorPaymentPosted", "DriverSettlementPosted"],
        RECOGNITION_BY_BASIS.Cash,
        RECOGNITION_BY_BASIS.Accrual,
      ),
    ).toEqual([
      "InvoicePosted",
      "CreditMemoPosted",
      "DebitMemoPosted",
      "VendorBillPosted",
      "DriverSettlementPosted",
    ]);
  });

  it("drops an invoice event chosen by hand once revenue moves to cash receipt", () => {
    expect(
      sourceEventsForRecognition(
        ["DebitMemoPosted"],
        RECOGNITION_BY_BASIS.Cash,
        RECOGNITION_BY_BASIS.Cash,
      ),
    ).toEqual(["CustomerPaymentPosted", "VendorPaymentPosted"]);
  });
});
