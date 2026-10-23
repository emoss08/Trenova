import type {
  DeskInvoice,
  DeskInvoiceChargeGroup,
  DeskInvoiceCredit,
  DeskInvoiceDispute,
  DeskInvoiceNeighbor,
  DeskInvoicePayment,
} from "@/lib/graphql/invoice-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** Midnight UTC of a day in 2026, as the server's dates read. */
export const DAY = {
  sep1: 1_788_220_800,
  sep2: 1_788_307_200,
  sep18: 1_789_689_600,
  sep19: 1_789_776_000,
  sep30: 1_790_726_400,
  oct1: 1_790_812_800,
  oct2: 1_790_899_200,
  oct3: 1_790_985_600,
  oct6: 1_791_244_800,
  oct9: 1_791_504_000,
  oct10: 1_791_590_400,
  oct14: 1_791_936_000,
  nov15: 1_794_700_800,
} as const;

export const HOUR = 3_600;

/** A translator that fills ICU arguments and plurals the way the catalog would. */
export const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

export const LONG_TEMPLATE =
  "Per mile – dry van, Texas triangle lanes with team surcharge and Q4 produce uplift";

export function chargeGroup(
  overrides: Partial<DeskInvoiceChargeGroup> = {},
): DeskInvoiceChargeGroup {
  return {
    type: "Accessorial",
    description: "Accessorial charge",
    chargeCode: null,
    chargeMethod: null,
    rateUnit: null,
    rate: null,
    rateBasisAmount: null,
    quantity: "1",
    formulaTemplateId: null,
    formulaTemplateName: null,
    lineCount: 1,
    shipmentCount: 1,
    amount: "0",
    ...overrides,
  };
}

export function payment(overrides: Partial<DeskInvoicePayment> = {}): DeskInvoicePayment {
  return {
    id: "cpa_ach",
    appliedAmountMinor: 285_000,
    shortPayAmountMinor: 81_612,
    createdAt: DAY.sep30 + 6 * HOUR,
    payment: {
      id: "cpm_ach",
      paymentDate: DAY.sep30,
      status: "Posted",
      paymentMethod: "ACH",
      referenceNumber: "884120",
    },
    ...overrides,
  };
}

export function credit(overrides: Partial<DeskInvoiceCredit> = {}): DeskInvoiceCredit {
  return {
    id: "cma_1",
    appliedAmountMinor: 22_500,
    accountingDate: DAY.oct9,
    status: "Applied",
    unappliedAt: null,
    unappliedReason: "",
    creditMemo: { id: "inv_cm31", number: "CM-2610-0031" },
    invoice: { id: "inv_412", number: "INV-2610-0412" },
    ...overrides,
  };
}

export function dispute(overrides: Partial<DeskInvoiceDispute> = {}): DeskInvoiceDispute {
  return {
    id: "idp_open",
    status: "Open",
    reasonCode: "AccessorialDisputed",
    disputedAmount: "185",
    notes: "Customer says their driver paid the lumper at the dock.",
    openedAt: DAY.oct6 + 10 * HOUR,
    resolvedAt: null,
    resolution: null,
    resolutionNotes: "",
    openedBy: { id: "usr_dana", name: "Dana Ruiz" },
    resolvedBy: null,
    ...overrides,
  };
}

export function neighbor(overrides: Partial<DeskInvoiceNeighbor> = {}): DeskInvoiceNeighbor {
  return {
    id: "inv_412",
    number: "INV-2610-0412",
    billType: "Invoice",
    status: "Posted",
    totalAmount: "3666.12",
    openBalance: "591.12",
    currencyCode: "USD",
    settlementStatus: "PartiallyPaid",
    daysPastDue: 12,
    postedAt: DAY.sep2,
    billToName: "Acme Foods",
    ...overrides,
  };
}

/**
 * A posted invoice for one shipment: partly paid by ACH (a check before it
 * bounced), credited once, twelve days past due with $185 of it in dispute,
 * sent by email and as an accepted 210.
 */
export function postedInvoice(overrides: Partial<DeskInvoice> = {}): DeskInvoice {
  return {
    id: "inv_412",
    number: "INV-2610-0412",
    billType: "Invoice",
    status: "Posted",
    scope: "Shipment",
    currencyCode: "USD",
    paymentTerm: "Net30",
    invoiceDate: DAY.sep2,
    dueDate: DAY.oct2,
    postedAt: DAY.sep2 + 9 * HOUR,
    periodStart: null,
    periodEnd: null,
    shipmentCount: 1,
    shipmentProNumber: "S-240193",
    shipmentBol: "BOL-55120",
    orderId: null,
    orderNumber: null,
    billToName: "Acme Foods",
    isSplitBill: false,
    factoringCompanyId: null,
    subtotalAmount: "2850",
    otherAmount: "816.12",
    totalAmount: "3666.12",
    appliedAmount: "3075",
    openBalance: "591.12",
    creditRemaining: "0",
    daysPastDue: 12,
    settlementStatus: "PartiallyPaid",
    disputeStatus: "Disputed",
    sendStatus: "Sent",
    sentAt: DAY.sep2 + 9 * HOUR,
    voidedAt: null,
    voidReason: "",
    voidDisposition: null,
    memoReason: "",
    memoKind: null,
    ratingNotes: [],
    ediSendStatus: "Sent",
    ediSentAt: DAY.sep2 + 9 * HOUR,
    pdfDocumentId: "doc_inv_412",
    lastEdiError: "",
    lastEdiMessageId: "edim_210",
    createdAt: DAY.sep2 + 9 * HOUR,
    updatedAt: DAY.oct9,
    customer: { id: "cus_acme", code: "ACME", name: "Acme Foods" },
    voidedBy: null,
    referenceInvoice: null,
    relatedInvoices: [],
    chargeSummary: [
      chargeGroup({
        type: "Freight",
        description: "Linehaul",
        quantity: "760",
        rate: "3.75",
        rateUnit: "$/mi",
        formulaTemplateId: "ft_permile",
        formulaTemplateName: LONG_TEMPLATE,
        amount: "2850",
      }),
      chargeGroup({
        description: "Fuel surcharge",
        chargeCode: "FSC",
        chargeMethod: "Percentage",
        rate: "14.25",
        rateBasisAmount: "2850",
        amount: "406.12",
      }),
      chargeGroup({
        description: "Detention",
        chargeCode: "DET",
        chargeMethod: "PerUnit",
        rateUnit: "Hour",
        rate: "75",
        quantity: "3",
        amount: "225",
      }),
      chargeGroup({
        description: "Lumper",
        chargeCode: "LUMP",
        chargeMethod: "Flat",
        rate: "185",
        amount: "185",
      }),
    ],
    paymentApplications: [
      payment(),
      payment({
        id: "cpa_check",
        appliedAmountMinor: 366_612,
        shortPayAmountMinor: 0,
        createdAt: DAY.sep18,
        payment: {
          id: "cpm_check",
          paymentDate: DAY.sep18,
          status: "Reversed",
          paymentMethod: "Check",
          referenceNumber: "1042",
        },
      }),
    ],
    creditApplications: [credit()],
    disputes: [
      dispute(),
      dispute({
        id: "idp_rate",
        status: "Resolved",
        reasonCode: "RateDiscrepancy",
        disputedAmount: "225",
        notes: "",
        openedAt: DAY.oct3,
        resolvedAt: DAY.oct9,
        resolution: "CreditIssued",
        resolutionNotes: "Detention hours did not match the gate log",
        resolvedBy: { id: "usr_dana", name: "Dana Ruiz" },
      }),
    ],
    ediSendPlan: {
      enabled: true,
      partnerName: "Acme Foods",
      communicationMethod: "AS2",
      status: "Sent",
      lastMessageId: "edim_210",
      lastError: "",
      sentAt: DAY.sep2 + 9 * HOUR,
      blockers: [],
    },
    ...overrides,
  };
}

/** A consolidated draft for Tyson: twelve shipments over a fortnight, its 210 blocked. */
export function consolidatedDraft(overrides: Partial<DeskInvoice> = {}): DeskInvoice {
  return postedInvoice({
    id: "inv_457",
    number: "INV-2610-0457",
    status: "Draft",
    scope: "Consolidated",
    paymentTerm: "Net45",
    invoiceDate: DAY.oct1,
    dueDate: DAY.nov15,
    postedAt: null,
    periodStart: DAY.oct1,
    periodEnd: DAY.oct14,
    shipmentCount: 12,
    shipmentProNumber: "",
    shipmentBol: "",
    billToName: "Tyson Foods",
    subtotalAmount: "15960",
    otherAmount: "2460",
    totalAmount: "18420",
    appliedAmount: "0",
    openBalance: "0",
    daysPastDue: null,
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    sentAt: null,
    ediSendStatus: "NotSent",
    ediSentAt: null,
    pdfDocumentId: null,
    lastEdiMessageId: null,
    customer: { id: "cus_tyson", code: "TYSON", name: "Tyson Foods" },
    chargeSummary: [
      chargeGroup({
        type: "Freight",
        description: "Linehaul",
        formulaTemplateId: "ft_permile",
        formulaTemplateName: "Per mile – dry van",
        lineCount: 8,
        shipmentCount: 8,
        quantity: "8",
        amount: "10920",
      }),
      chargeGroup({
        type: "Freight",
        description: "Linehaul",
        formulaTemplateId: "ft_cwt",
        formulaTemplateName: "Per CWT (Hundredweight)",
        rate: "3.05",
        rateUnit: "$/cwt",
        lineCount: 4,
        shipmentCount: 4,
        quantity: "1652.46",
        amount: "5040",
      }),
      chargeGroup({
        description: "Fuel surcharge",
        chargeCode: "FSC",
        chargeMethod: "Percentage",
        rate: "11.5",
        lineCount: 12,
        shipmentCount: 12,
        quantity: "12",
        amount: "1830",
      }),
      chargeGroup({
        description: "Detention",
        chargeCode: "DET",
        chargeMethod: "PerUnit",
        rateUnit: "Hour",
        rate: "75",
        quantity: "6",
        lineCount: 4,
        shipmentCount: 4,
        amount: "450",
      }),
    ],
    paymentApplications: [],
    creditApplications: [],
    disputes: [],
    ediSendPlan: {
      enabled: true,
      partnerName: "Tyson Foods",
      communicationMethod: "",
      status: "NotConfigured",
      lastMessageId: null,
      lastError: "",
      sentAt: null,
      blockers: ["Tyson Foods has no 210 document profile."],
    },
    ...overrides,
  });
}

/** Voided after posting: reversed in full, the freight released to rebill. */
export function voidedInvoice(overrides: Partial<DeskInvoice> = {}): DeskInvoice {
  return postedInvoice({
    id: "inv_388",
    number: "INV-2610-0388",
    status: "Voided",
    openBalance: "0",
    appliedAmount: "0",
    totalAmount: "2140",
    subtotalAmount: "1850",
    otherAmount: "290",
    daysPastDue: null,
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    voidedAt: DAY.oct3 + 11 * HOUR,
    voidReason: "Billed to Acme, but the consignee pays on this lane.",
    voidDisposition: "Rebill",
    voidedBy: { id: "usr_mia", name: "Mia Chen" },
    paymentApplications: [],
    creditApplications: [],
    disputes: [],
    ...overrides,
  });
}

/** A credit memo correcting INV-2610-0412: $400 of credit, $225 applied, one use taken back. */
export function creditMemo(overrides: Partial<DeskInvoice> = {}): DeskInvoice {
  return postedInvoice({
    id: "inv_cm31",
    number: "CM-2610-0031",
    billType: "CreditMemo",
    scope: "Memo",
    shipmentCount: 0,
    shipmentProNumber: "",
    shipmentBol: "",
    subtotalAmount: "0",
    otherAmount: "-400",
    totalAmount: "-400",
    appliedAmount: "0",
    openBalance: "0",
    creditRemaining: "175",
    daysPastDue: null,
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    memoReason: "Detention on S-240193 did not match the gate log.",
    memoKind: "Manual",
    referenceInvoice: neighbor(),
    chargeSummary: [
      chargeGroup({
        type: "Memo",
        description: "Detention credit",
        amount: "-225",
        shipmentCount: 0,
      }),
      chargeGroup({
        type: "Memo",
        description: "Rate correction",
        amount: "-175",
        shipmentCount: 0,
      }),
    ],
    paymentApplications: [],
    creditApplications: [
      credit(),
      credit({
        id: "cma_2",
        appliedAmountMinor: 17_500,
        status: "Unapplied",
        unappliedAt: DAY.oct10,
        unappliedReason: "Applied to the wrong invoice",
        invoice: { id: "inv_401", number: "INV-2610-0401" },
      }),
    ],
    disputes: [],
    ...overrides,
  });
}
