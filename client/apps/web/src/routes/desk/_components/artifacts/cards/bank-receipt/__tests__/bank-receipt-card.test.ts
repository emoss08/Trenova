import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { cardDay, minorMoney } from "../../../record-card/record-card-values";
import {
  applicationLine,
  candidateEvidence,
  candidateName,
  candidateOpen,
  candidateStanding,
  candidatesWithheldNote,
  paymentLine,
  paymentStats,
  receiptCheck,
  receiptFacts,
  receiptMemo,
  receiptStanding,
  receiptTitle,
  workItemHolder,
  workItemStanding,
} from "../bank-receipt-card";
import { MATCHED_AT, RECEIVED, candidate, exceptionCard, matchedCard, payment } from "./fixtures";

/** Fills a message's numbered places the way the catalog would. */
const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

describe("bank receipt card", () => {
  it("stands each status in the reconciliation page's words and colours", () => {
    expect(receiptStanding("Exception", t)).toEqual({ label: "Exception", tone: "danger" });
    expect(receiptStanding("Matched", t)).toEqual({ label: "Matched", tone: "success" });
    expect(receiptStanding("Imported", t)).toEqual({ label: "Imported", tone: "muted" });
  });

  it("names a receipt by the bank's reference, and one with none as such", () => {
    expect(receiptTitle(exceptionCard(), t)).toBe("ACH 7731090");
    const bare = exceptionCard();
    bare.receipt = { ...bare.receipt, referenceNumber: "  " };
    expect(receiptTitle(bare, t)).toBe("Receipt with no reference");
    expect(receiptMemo(exceptionCard())).toBe("ACH CREDIT KESSLER FARMS");
    expect(receiptMemo(matchedCard())).toBeUndefined();
  });

  it("says when the money came in, and when it was matched only once it was", () => {
    expect(receiptFacts(exceptionCard(), t)).toEqual([`Received ${cardDay(RECEIVED)}`, ""]);
    expect(receiptFacts(matchedCard(), t)).toEqual([
      `Received ${cardDay(RECEIVED)}`,
      `Matched ${cardDay(MATCHED_AT)}`,
    ]);
  });

  it("says why an exception is not applied in the product's own reason, and never that it matches", () => {
    const check = receiptCheck(exceptionCard(), t);
    expect(check).toEqual({
      state: "bad",
      title: "No posted payment matches it",
      detail: "No unique customer payment match found for bank receipt",
    });
    const bare = exceptionCard();
    bare.receipt = { ...bare.receipt, exceptionReason: "" };
    expect(receiptCheck(bare, t).detail).toBe(
      "It was not matched to a posted customer payment when it was imported.",
    );
  });

  it("says a matched receipt is applied through its payer's payment, or that the payment is not the reader's", () => {
    expect(receiptCheck(matchedCard(), t)).toEqual({
      state: "ok",
      title: "Applied through a posted payment",
      detail: `Matched to Northline Grocers's payment of ${minorMoney(1_824_000)}`,
    });
    expect(receiptCheck(matchedCard({ payment: null, paymentHidden: true }), t).detail).toBe(
      "It is matched to a customer payment you may not read.",
    );
    expect(receiptCheck(matchedCard({ payment: null }), t).detail).toBe(
      "The payment it was matched to is gone.",
    );
    const imported = exceptionCard();
    imported.receipt = { ...imported.receipt, status: "Imported" };
    expect(receiptCheck(imported, t)).toMatchObject({ state: "info", title: "Not reconciled yet" });
  });

  it("writes the payment's method in words with its reference and day, and warns of what is not applied", () => {
    expect(paymentLine(payment(), t)).toBe(`ACH 7731042, paid ${cardDay(RECEIVED)}`);
    expect(paymentLine(payment({ referenceNumber: "", paymentMethod: "Other" }), t)).toBe(
      `Payment, paid ${cardDay(RECEIVED)}`,
    );
    expect(paymentStats(payment(), t)).toEqual([
      { key: "amount", label: "Payment", value: minorMoney(1_824_000) },
      { key: "applied", label: "Applied", value: minorMoney(1_800_000) },
      { key: "unapplied", label: "Not applied", value: minorMoney(24_000), state: "warn" },
    ]);
    expect(paymentStats(payment({ unappliedAmountMinor: 0 }), t)[2]?.state).toBeUndefined();
  });

  it("names each invoice the payment went to, its short pay, and one the reader may not read", () => {
    const [first, second, hidden] = payment().applications;
    expect(applicationLine(first!, "USD", t)).toEqual({
      name: "INV2610000812",
      amount: minorMoney(1_200_000),
      note: "",
    });
    expect(applicationLine(second!, "USD", t).note).toBe(`${minorMoney(4_250)} short`);
    expect(applicationLine(hidden!, "USD", t).name).toBe("An invoice you may not read");
  });

  it("stands the queue entry in the queue's words and says who holds it", () => {
    const item = exceptionCard().workItem!;
    expect(workItemStanding(item, t)).toEqual({ label: "Assigned", tone: "progress" });
    expect(workItemStanding({ ...item, status: "InReview" }, t)).toEqual({
      label: "In review",
      tone: "warn",
    });
    expect(workItemHolder(item, t)).toBe(`Mia Chen since ${cardDay(RECEIVED + 3600)}`);
    expect(workItemHolder({ ...item, assignedTo: "" }, t)).toBe("Nobody has taken it yet");
  });
});

describe("bank receipt candidates", () => {
  it("says to the cent only for an exact standing, and the gap either side of it", () => {
    expect(candidateStanding(candidate({ standing: "Exact", differenceMinor: 0 }), t)).toEqual({
      label: "To the cent",
      tone: "success",
    });
    expect(candidateStanding(candidate(), t)).toEqual({
      label: `${minorMoney(12_000)} short`,
      tone: "warn",
    });
    expect(candidateStanding(candidate({ standing: "Over", differenceMinor: 4_510 }), t)).toEqual({
      label: `${minorMoney(4_510)} over`,
      tone: "warn",
    });
    expect(
      candidateStanding(candidate({ standing: "SameAmount", differenceMinor: 0 }), t).label,
    ).toBe("Same amount");
  });

  it("names an invoice by its number and a payment by its customer and reference", () => {
    expect(candidateName(candidate(), t)).toBe("SEED-INV-0003");
    expect(candidateName(candidate({ invoiceNumber: "" }), t)).toBe("An invoice you may not read");
    expect(
      candidateName(
        candidate({ kind: "CustomerPayment", paymentReference: "ACH-1", invoiceNumber: "" }),
        t,
      ),
    ).toBe("Premier Farms's payment ACH-1");
    expect(candidateName(candidate({ kind: "CustomerPayment", invoiceNumber: "" }), t)).toBe(
      "Premier Farms's payment",
    );
  });

  it("words each piece of evidence and the open amount, and skips a code it does not know", () => {
    expect(
      candidateEvidence(
        candidate({ evidence: ["NumberInMemo", "AmountExact", "Payer", "New"] }),
        t,
      ),
    ).toEqual([
      "The bank's memo names it",
      "Its open amount is the receipt's",
      "Its customer sent a posted payment like this receipt",
    ]);
    expect(candidateOpen(candidate(), t)).toBe(`${minorMoney(427_500)} open`);
    expect(candidateOpen(candidate({ kind: "CustomerPayment" }), t)).toBe(
      `${minorMoney(427_500)} paid`,
    );
  });

  it("says which kinds of candidate are withheld from the reader", () => {
    expect(candidatesWithheldNote(exceptionCard(), t)).toBe("");
    expect(candidatesWithheldNote(exceptionCard({ candidatesWithheld: ["Invoice"] }), t)).toBe(
      "You may not read invoices, so invoices it could pay are not shown.",
    );
    expect(
      candidatesWithheldNote(
        exceptionCard({ candidatesWithheld: ["Invoice", "CustomerPayment"] }),
        t,
      ),
    ).toBe("You may not read invoices or customer payments, so none are shown.");
  });
});
