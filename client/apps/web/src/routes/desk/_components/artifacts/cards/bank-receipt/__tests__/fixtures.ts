import type {
  BankReceiptCard,
  BankReceiptCardCandidate,
  BankReceiptCardPayment,
} from "@/types/bank-receipt";

/** 2026-11-20 15:00 UTC: when the receipts in these fixtures came into the bank. */
export const RECEIVED = Date.UTC(2026, 10, 20, 15) / 1000;
export const MATCHED_AT = Date.UTC(2026, 10, 21, 16) / 1000;

/** A receipt the import could not match, written from the Go `bankreceiptports.DeskCard` contract. */
export function exceptionCard(overrides: Partial<BankReceiptCard> = {}): BankReceiptCard {
  return {
    receipt: {
      id: "brcpt_1",
      receiptDate: RECEIVED,
      amountMinor: 945_000,
      referenceNumber: "ACH 7731090",
      memo: "ACH CREDIT KESSLER FARMS",
      status: "Exception",
      matchedCustomerPaymentId: "",
      matchedAt: null,
      matchedById: "",
      exceptionReason: "No unique customer payment match found for bank receipt",
      importBatchId: "brib_1",
    },
    payment: null,
    paymentHidden: false,
    workItem: {
      id: "brwi_1",
      status: "Assigned",
      assignedTo: "Mia Chen",
      assignedAt: RECEIVED + 3600,
      resolutionType: "",
      resolutionNote: "",
    },
    workItemShown: true,
    candidates: [],
    candidatesWithheld: [],
    ...overrides,
  };
}

/** An open invoice the receipt's reference names, $120.00 more than the receipt. */
export function candidate(
  overrides: Partial<BankReceiptCardCandidate> = {},
): BankReceiptCardCandidate {
  return {
    rank: 1,
    kind: "Invoice",
    invoiceId: "inv_9",
    invoiceNumber: "SEED-INV-0003",
    paymentId: "",
    paymentReference: "",
    customerName: "Premier Farms",
    standing: "Short",
    evidence: ["NumberInReference"],
    openMinor: 427_500,
    differenceMinor: -12_000,
    ...overrides,
  };
}

export function payment(overrides: Partial<BankReceiptCardPayment> = {}): BankReceiptCardPayment {
  return {
    id: "cpay_1",
    customerId: "cus_1",
    customerName: "Northline Grocers",
    paymentMethod: "ACH",
    referenceNumber: "7731042",
    paymentDate: RECEIVED,
    status: "Posted",
    currencyCode: "USD",
    amountMinor: 1_824_000,
    appliedAmountMinor: 1_800_000,
    unappliedAmountMinor: 24_000,
    applications: [
      {
        invoiceId: "inv_1",
        invoiceNumber: "INV2610000812",
        appliedAmountMinor: 1_200_000,
        shortPayAmountMinor: 0,
      },
      {
        invoiceId: "inv_2",
        invoiceNumber: "INV2610000813",
        appliedAmountMinor: 600_000,
        shortPayAmountMinor: 4_250,
      },
      {
        invoiceId: "inv_3",
        invoiceNumber: "",
        appliedAmountMinor: 0,
        shortPayAmountMinor: 0,
      },
    ],
    ...overrides,
  };
}

/** A receipt matched to a posted payment, with no queue entry left open. */
export function matchedCard(overrides: Partial<BankReceiptCard> = {}): BankReceiptCard {
  const base = exceptionCard();
  return {
    ...base,
    receipt: {
      ...base.receipt,
      id: "brcpt_2",
      amountMinor: 1_824_000,
      referenceNumber: "ACH 7731042",
      memo: "",
      status: "Matched",
      matchedCustomerPaymentId: "cpay_1",
      matchedAt: MATCHED_AT,
      exceptionReason: "",
    },
    payment: payment(),
    workItem: null,
    ...overrides,
  };
}
