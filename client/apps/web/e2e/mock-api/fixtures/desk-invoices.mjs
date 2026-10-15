// The invoices the Desk cards thread names, as the DeskInvoiceCard query
// answers for them, each dated against the mock's now: one posted and past due
// with a payment, a credit and an open dispute; a consolidated draft waiting
// to be posted whose 210 cannot go; one posted today with nothing behind it; a
// voided one; and a credit memo that corrects the first.

const HOUR = 3_600;
const DAY = 86_400;

/** Midnight UTC `days` before the anchor, as the server dates an invoice. */
function dayBefore(anchor, days) {
  return Math.floor(anchor / DAY) * DAY - days * DAY;
}

function group(overrides) {
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

function neighbor(invoice) {
  return {
    id: invoice.id,
    number: invoice.number,
    billType: invoice.billType,
    status: invoice.status,
    totalAmount: invoice.totalAmount,
    openBalance: invoice.openBalance,
    currencyCode: invoice.currencyCode,
    settlementStatus: invoice.settlementStatus,
    daysPastDue: invoice.daysPastDue,
    postedAt: invoice.postedAt,
    billToName: invoice.billToName,
  };
}

const NO_EDI = (partnerName) => ({
  enabled: false,
  partnerName,
  communicationMethod: "",
  status: "NotSent",
  lastMessageId: null,
  lastError: "",
  sentAt: null,
  blockers: [],
});

function base(overrides) {
  return {
    billType: "Invoice",
    status: "Posted",
    scope: "Shipment",
    currencyCode: "USD",
    paymentTerm: "Net30",
    periodStart: null,
    periodEnd: null,
    shipmentCount: 1,
    shipmentProNumber: "",
    shipmentBol: "",
    orderId: null,
    orderNumber: null,
    isSplitBill: false,
    factoringCompanyId: null,
    subtotalAmount: "0",
    otherAmount: "0",
    totalAmount: "0",
    appliedAmount: "0",
    openBalance: "0",
    creditRemaining: "0",
    daysPastDue: null,
    settlementStatus: "Unpaid",
    disputeStatus: "None",
    sendStatus: "NotSent",
    sentAt: null,
    voidedAt: null,
    voidReason: "",
    voidDisposition: null,
    memoReason: "",
    memoKind: null,
    ediSendStatus: "NotSent",
    ediSentAt: null,
    lastEdiError: "",
    lastEdiMessageId: null,
    voidedBy: null,
    referenceInvoice: null,
    relatedInvoices: [],
    chargeSummary: [],
    paymentApplications: [],
    creditApplications: [],
    disputes: [],
    ...overrides,
  };
}

/** Each invoice the cards thread names, with what its card reads beside it. */
export function deskInvoices(anchor) {
  const issued = dayBefore(anchor, 42);
  const acme = base({
    id: "inv_2610_0042",
    number: "INV-2610-0042",
    invoiceDate: issued,
    dueDate: issued + 30 * DAY,
    postedAt: issued + 9 * HOUR,
    shipmentProNumber: "S-240193",
    shipmentBol: "BOL-88213",
    billToName: "Acme Foods",
    customer: { id: "cus_acme", code: "ACME", name: "Acme Foods" },
    subtotalAmount: "2350",
    otherAmount: "580.5",
    totalAmount: "2930.5",
    appliedAmount: "2425",
    openBalance: "505.5",
    daysPastDue: 12,
    settlementStatus: "PartiallyPaid",
    disputeStatus: "Disputed",
    sendStatus: "Sent",
    sentAt: issued + 9 * HOUR + 120,
    ediSendStatus: "Sent",
    ediSentAt: issued + 9 * HOUR + 180,
    lastEdiMessageId: "edimsg_210_acme",
    createdAt: issued + 8 * HOUR,
    updatedAt: anchor - 5 * DAY,
    chargeSummary: [
      group({
        type: "Freight",
        description: "Freight charge",
        rate: "3.25",
        formulaTemplateId: "ft_mile",
        formulaTemplateName:
          "Per mile – dry van, Texas triangle lanes with team surcharge and Q4 produce uplift",
        amount: "2350",
      }),
      group({
        description: "Fuel surcharge",
        chargeCode: "FSC",
        chargeMethod: "Percentage",
        rate: "11.51",
        rateBasisAmount: "2350",
        amount: "270.5",
      }),
      group({
        description: "Detention",
        chargeCode: "DET",
        chargeMethod: "PerUnit",
        rateUnit: "Hour",
        rate: "75",
        quantity: "3",
        amount: "225",
      }),
      group({
        description: "Lumper",
        chargeCode: "LUMP",
        chargeMethod: "Flat",
        rate: "85",
        amount: "85",
      }),
    ],
    paymentApplications: [
      {
        id: "cpa_ach",
        appliedAmountMinor: 220_000,
        shortPayAmountMinor: 73_050,
        createdAt: anchor - 14 * DAY,
        payment: {
          id: "cpm_ach",
          paymentDate: dayBefore(anchor, 14),
          status: "Posted",
          paymentMethod: "ACH",
          referenceNumber: "884120",
        },
      },
      {
        id: "cpa_check",
        appliedAmountMinor: 293_050,
        shortPayAmountMinor: 0,
        createdAt: anchor - 26 * DAY,
        payment: {
          id: "cpm_check",
          paymentDate: dayBefore(anchor, 26),
          status: "Reversed",
          paymentMethod: "Check",
          referenceNumber: "1042",
        },
      },
    ],
    creditApplications: [
      {
        id: "cma_1",
        appliedAmountMinor: 22_500,
        accountingDate: dayBefore(anchor, 5),
        status: "Applied",
        unappliedAt: null,
        unappliedReason: "",
        creditMemo: { id: "inv_2610_cm31", number: "CM-2610-0031" },
        invoice: { id: "inv_2610_0042", number: "INV-2610-0042" },
      },
    ],
    disputes: [
      {
        id: "idp_lumper",
        status: "Open",
        reasonCode: "AccessorialDisputed",
        disputedAmount: "85",
        notes:
          "Customer says their driver paid the lumper at the dock. Asked for the receipt from Atlanta DC.",
        openedAt: anchor - 8 * DAY - 5 * HOUR,
        resolvedAt: null,
        resolution: null,
        resolutionNotes: "",
        openedBy: { id: "usr_dana", name: "Dana Ruiz" },
        resolvedBy: null,
      },
      {
        id: "idp_detention",
        status: "Resolved",
        reasonCode: "RateDiscrepancy",
        disputedAmount: "225",
        notes: "",
        openedAt: anchor - 11 * DAY,
        resolvedAt: anchor - 5 * DAY,
        resolution: "CreditIssued",
        resolutionNotes: "Detention hours did not match the gate log",
        openedBy: { id: "usr_dana", name: "Dana Ruiz" },
        resolvedBy: { id: "usr_dana", name: "Dana Ruiz" },
      },
    ],
    ediSendPlan: {
      enabled: true,
      partnerName: "Acme Foods",
      communicationMethod: "AS2",
      status: "Sent",
      lastMessageId: "edimsg_210_acme",
      lastError: "",
      sentAt: issued + 9 * HOUR + 180,
      blockers: [],
    },
  });

  const drafted = dayBefore(anchor, 13);
  const tyson = base({
    id: "inv_2610_0457",
    number: "INV-2610-0457",
    status: "Draft",
    scope: "Consolidated",
    paymentTerm: "Net45",
    invoiceDate: drafted,
    dueDate: drafted + 45 * DAY,
    postedAt: null,
    periodStart: drafted,
    periodEnd: dayBefore(anchor, 0),
    shipmentCount: 12,
    billToName: "Tyson Foods",
    customer: { id: "cus_tyson", code: "TYSON", name: "Tyson Foods" },
    subtotalAmount: "15960",
    otherAmount: "2460",
    totalAmount: "18420",
    createdAt: drafted + 7 * HOUR,
    updatedAt: anchor - 2 * DAY,
    chargeSummary: [
      group({
        type: "Freight",
        description: "Freight charge",
        formulaTemplateId: "ft_mile",
        formulaTemplateName: "Per mile – dry van",
        lineCount: 8,
        shipmentCount: 8,
        quantity: "8",
        amount: "10920",
      }),
      group({
        type: "Freight",
        description: "Freight charge",
        formulaTemplateId: "ft_flat",
        formulaTemplateName: "Per CWT (Hundredweight)",
        rate: "3.05",
        lineCount: 4,
        shipmentCount: 4,
        quantity: "4",
        amount: "5040",
      }),
      group({
        description: "Fuel surcharge",
        chargeCode: "FSC",
        chargeMethod: "Percentage",
        rate: "11.5",
        lineCount: 12,
        shipmentCount: 12,
        quantity: "12",
        amount: "1830",
      }),
      group({
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
      group({
        description: "Lumper",
        chargeCode: "LUMP",
        chargeMethod: "Flat",
        rate: "60",
        quantity: "3",
        lineCount: 3,
        shipmentCount: 3,
        amount: "180",
      }),
    ],
    ediSendPlan: {
      enabled: true,
      partnerName: "Tyson Foods",
      communicationMethod: "",
      status: "NotConfigured",
      lastMessageId: null,
      lastError: "",
      sentAt: null,
      blockers: [
        "Tyson Foods has no 210 document profile. Add one on the partner before posting, or it will go by email only.",
      ],
    },
  });

  const today = dayBefore(anchor, 0);
  const riverside = base({
    id: "inv_2610_0463",
    number: "INV-2610-0463",
    invoiceDate: today,
    dueDate: today + 30 * DAY,
    postedAt: anchor - 2 * HOUR,
    shipmentProNumber: "S-240340",
    shipmentBol: "",
    billToName: "Riverside Grocers",
    customer: { id: "cus_riverside", code: "RIVER", name: "Riverside Grocers" },
    subtotalAmount: "1280",
    totalAmount: "1280",
    openBalance: "1280",
    createdAt: anchor - 2 * HOUR,
    updatedAt: anchor - 2 * HOUR,
    chargeSummary: [
      group({
        type: "Freight",
        description: "Freight charge",
        rate: "3.05",
        formulaTemplateId: "ft_flat",
        formulaTemplateName: "Per CWT (Hundredweight)",
        amount: "1280",
      }),
    ],
    ediSendPlan: NO_EDI("Riverside Grocers"),
  });

  const voidedOn = dayBefore(anchor, 11);
  const voided = base({
    id: "inv_2610_0388",
    number: "INV-2610-0388",
    status: "Voided",
    invoiceDate: dayBefore(anchor, 20),
    dueDate: dayBefore(anchor, 20) + 30 * DAY,
    postedAt: dayBefore(anchor, 20) + 8 * HOUR,
    shipmentProNumber: "S-240150",
    billToName: "Acme Foods",
    customer: { id: "cus_acme", code: "ACME", name: "Acme Foods" },
    subtotalAmount: "1850",
    otherAmount: "290",
    totalAmount: "2140",
    sendStatus: "Sent",
    sentAt: dayBefore(anchor, 20) + 8 * HOUR + 120,
    voidedAt: voidedOn + 11 * HOUR,
    voidReason:
      "Billed to Acme, but the consignee pays on this lane. Rebilling to Riverside Grocers.",
    voidDisposition: "Rebill",
    voidedBy: { id: "usr_mia", name: "Mia Chen" },
    createdAt: dayBefore(anchor, 20) + 7 * HOUR,
    updatedAt: voidedOn + 11 * HOUR,
    chargeSummary: [
      group({
        type: "Freight",
        description: "Freight charge",
        formulaTemplateId: "ft_flat",
        formulaTemplateName: "Flat lane rate",
        amount: "1850",
      }),
      group({
        description: "Fuel surcharge",
        chargeCode: "FSC",
        chargeMethod: "Percentage",
        rate: "11.95",
        rateBasisAmount: "1850",
        amount: "221",
      }),
      group({
        description: "Stop off",
        chargeCode: "STOP",
        chargeMethod: "Flat",
        rate: "69",
        amount: "69",
      }),
    ],
    ediSendPlan: NO_EDI("Acme Foods"),
  });

  const credit = base({
    id: "inv_2610_cm31",
    number: "CM-2610-0031",
    billType: "CreditMemo",
    scope: "Memo",
    shipmentCount: 0,
    invoiceDate: dayBefore(anchor, 5),
    dueDate: null,
    postedAt: anchor - 5 * DAY + 14 * HOUR,
    billToName: "Acme Foods",
    customer: { id: "cus_acme", code: "ACME", name: "Acme Foods" },
    otherAmount: "-400",
    totalAmount: "-400",
    creditRemaining: "175",
    memoReason:
      "Detention on S-240193 did not match the gate log at Atlanta DC, and S-240170 was rated at last year's contract.",
    memoKind: "Manual",
    sendStatus: "Sent",
    sentAt: anchor - 5 * DAY + 14 * HOUR + 60,
    referenceInvoice: neighbor(acme),
    createdAt: anchor - 5 * DAY + 13 * HOUR,
    updatedAt: anchor - 4 * DAY,
    chargeSummary: [
      group({ type: "Memo", description: "Detention credit", amount: "-225", shipmentCount: 0 }),
      group({ type: "Memo", description: "Rate correction", amount: "-175", shipmentCount: 0 }),
    ],
    creditApplications: [
      {
        id: "cma_1",
        appliedAmountMinor: 22_500,
        accountingDate: dayBefore(anchor, 5),
        status: "Applied",
        unappliedAt: null,
        unappliedReason: "",
        creditMemo: { id: "inv_2610_cm31", number: "CM-2610-0031" },
        invoice: { id: "inv_2610_0042", number: "INV-2610-0042" },
      },
      {
        id: "cma_2",
        appliedAmountMinor: 17_500,
        accountingDate: dayBefore(anchor, 5),
        status: "Unapplied",
        unappliedAt: dayBefore(anchor, 4),
        unappliedReason: "Applied to the wrong invoice",
        creditMemo: { id: "inv_2610_cm31", number: "CM-2610-0031" },
        invoice: { id: "inv_2610_0388", number: "INV-2610-0388" },
      },
    ],
    ediSendPlan: NO_EDI("Acme Foods"),
  });

  const sync = (invoice, provider, docNumber, at) => ({
    objectType: invoice.billType === "CreditMemo" ? "CreditMemo" : "Invoice",
    objectId: invoice.id,
    providerName: provider,
    record: {
      id: `acctsr_${invoice.id}`,
      objectType: invoice.billType === "CreditMemo" ? "CreditMemo" : "Invoice",
      objectId: invoice.id,
      objectNumber: invoice.number,
      operation: invoice.status === "Voided" ? "Void" : "Create",
      sourceEvent: "InvoicePosted",
      revision: 1,
      documentDate: invoice.invoiceDate,
      dependsOnRecordId: null,
      status: "Synced",
      approvalRequired: false,
      attemptCount: 1,
      maxAttempts: 8,
      nextAttemptAt: null,
      lastAttemptAt: at,
      syncedAt: at,
      externalId: docNumber,
      externalDocNumber: docNumber,
      externalUrl: `https://accounting.example.com/${invoice.number}`,
      errorCategory: null,
      errorCode: "",
      errorMessage: "",
      resolution: "",
      skippedReason: "",
      skippedAt: null,
      skippedById: null,
      createdAt: at,
      updatedAt: at,
    },
  });

  const entry = (id, operation, hoursAgo, name, changes = null, comment = null) => ({
    id,
    operation,
    timestamp: anchor - hoursAgo * HOUR,
    changes,
    comment,
    principalType: name ? "session_user" : "system",
    user: name ? { id: `usr_${id}`, name } : null,
  });

  return {
    [acme.id]: {
      invoice: acme,
      sync: sync(acme, "QuickBooks Online", "1042", issued + 9 * HOUR + 300),
      attempt: { toRecipients: ["ap@acmefoods.com"], sentAt: acme.sentAt, error: null },
      plan: null,
      activity: [
        entry("ae_i1", "update", 5 * 24 - 2, "Dana Ruiz", null, "Applied credit memo CM-2610-0031"),
        entry("ae_i2", "update", 8 * 24 + 5, "Dana Ruiz", {
          disputeStatus: { from: "None", to: "Disputed" },
        }),
        entry("ae_i3", "update", 14 * 24, null, {
          settlementStatus: { from: "Unpaid", to: "PartiallyPaid" },
        }),
        entry("ae_i4", "update", 42 * 24 - 9, "Mia Chen", {
          status: { from: "Draft", to: "Posted" },
        }),
        entry("ae_i5", "create", 42 * 24 - 8, "Mia Chen"),
      ],
    },
    [tyson.id]: {
      invoice: tyson,
      sync: null,
      attempt: null,
      plan: { to: ["ap@tyson.com"], errors: [] },
      activity: [
        entry(
          "ae_t1",
          "create",
          13 * 24 - 7,
          null,
          null,
          "Built from the October 1–14 statement run",
        ),
      ],
    },
    [riverside.id]: {
      invoice: riverside,
      sync: null,
      attempt: null,
      plan: { to: [], errors: ["Riverside Grocers has no billing email on file."] },
      activity: [entry("ae_r1", "create", 2, "Mia Chen")],
    },
    [voided.id]: {
      invoice: voided,
      sync: sync(voided, "Xero", "INV-0388", voidedOn + 11 * HOUR + 600),
      attempt: { toRecipients: ["ap@acmefoods.com"], sentAt: voided.sentAt, error: null },
      plan: null,
      activity: [
        entry("ae_v1", "update", 11 * 24 - 11, "Mia Chen", {
          status: { from: "Posted", to: "Voided" },
        }),
        entry("ae_v2", "create", 20 * 24 - 7, "Mia Chen"),
      ],
    },
    [credit.id]: {
      invoice: credit,
      sync: sync(credit, "QuickBooks Online", "1057", credit.postedAt + 300),
      attempt: { toRecipients: ["ap@acmefoods.com"], sentAt: credit.sentAt, error: null },
      plan: null,
      activity: [entry("ae_c1", "create", 5 * 24 - 13, "Dana Ruiz")],
    },
  };
}

/** The cards thread's invoice conversations: each slug's invoice, its title, and what was asked and said. */
export const DESK_INVOICE_CONVERSATIONS = {
  "invoice-acme": {
    recordId: "inv_2610_0042",
    title: "Invoice INV-2610-0042",
    question: "Where are we on Acme's INV-2610-0042?",
    reply:
      "It's 12 days past due with $505.50 still open. Acme short-paid the lumper and has it in dispute. [INV-2610-0042](artifact:art_invoice-acme) has the payments and the dispute.",
  },
  "invoice-tyson-draft": {
    recordId: "inv_2610_0457",
    title: "Invoice INV-2610-0457",
    question: "Why hasn't Tyson's October invoice gone out?",
    reply:
      "It's still a draft, unposted for 13 days, and its 210 can't go because Tyson has no 210 profile. [INV-2610-0457](artifact:art_invoice-tyson-draft) shows the charges and where it will go.",
  },
  "invoice-riverside": {
    recordId: "inv_2610_0463",
    title: "Invoice INV-2610-0463",
    question: "Did Riverside's invoice go out?",
    reply:
      "It posted this morning, but Riverside has no billing email on file, so nothing was sent. [INV-2610-0463](artifact:art_invoice-riverside) has the details.",
  },
  "invoice-voided": {
    recordId: "inv_2610_0388",
    title: "Invoice INV-2610-0388",
    question: "What happened to INV-2610-0388?",
    reply:
      "Mia voided it: the consignee pays on that lane, so the freight went back to the billing queue to rebill. [INV-2610-0388](artifact:art_invoice-voided) keeps why.",
  },
  "invoice-credit": {
    recordId: "inv_2610_cm31",
    title: "Credit memo CM-2610-0031",
    question: "How much of Acme's credit memo is left?",
    reply:
      "$175 of the $400 is left; one use was taken back after it went to the wrong invoice. [CM-2610-0031](artifact:art_invoice-credit) lists where it went.",
  },
  "invoice-gone": {
    recordId: "inv_2608_0001",
    title: "Invoice INV-2608-0001",
    question: "Pull up INV-2608-0001.",
    reply:
      "That invoice has been removed since I looked it up. [INV-2608-0001](artifact:art_invoice-gone) is what I found then.",
  },
};
