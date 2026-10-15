// The worker and carrier settlements the Desk cards thread names, as the
// DeskDriverSettlementCard and DeskCarrierSettlementCard queries answer for
// them, each dated against the mock's now. Workers: a company worker's week
// waiting on approval with flags and a dispute; an owner-operator whose
// deductions ran past earnings; one paid on its pay date; an empty draft; one
// voided. Carriers: a week waiting on approval with an invoice over the buy
// rate and a load not invoiced; one approved with an invoice waiting to be
// accepted; one paid by check; an empty draft; one voided.

const HOUR = 3_600;
const DAY = 86_400;

/** Midnight UTC `days` before the anchor (after it, when negative), as the server dates a settlement. */
function dayBefore(anchor, days) {
  return Math.floor(anchor / DAY) * DAY - days * DAY;
}

const user = (id, name) => ({ id, name });
const MIA = user("usr_mia", "Mia Chen");
const DANA = user("usr_dana", "Dana Ruiz");
const LI = user("usr_li", "Li Wei");

function entryAt(anchor) {
  return (id, operation, hoursAgo, name, changes = null, comment = null) => ({
    id,
    operation,
    timestamp: anchor - hoursAgo * HOUR,
    changes,
    comment,
    principalType: name ? "session_user" : "system",
    user: name ? { id: `usr_${id}`, name } : null,
  });
}

function payGroup(overrides) {
  return {
    category: "Earning",
    componentKind: null,
    method: null,
    description: "",
    rate: null,
    quantity: "0",
    amountMinor: 0,
    lineCount: 1,
    shipmentCount: 0,
    advanceId: null,
    escrowAccountId: null,
    recurringDeductionId: null,
    manual: false,
    advance: null,
    escrowAccount: null,
    ...overrides,
  };
}

/** The worker settlements, keyed by ID, each with its activity. */
export function deskDriverSettlements(anchor) {
  const entry = entryAt(anchor);
  const start = dayBefore(anchor, 10);
  const end = dayBefore(anchor, 4);
  const life = {
    periodStart: start,
    periodEnd: end,
    payDate: dayBefore(anchor, -2),
    currencyCode: "USD",
    approvedAt: null,
    postedAt: null,
    paidAt: null,
    paymentMethod: "",
    paymentReference: "",
    voidedAt: null,
    voidReason: "",
    approvedBy: null,
    postedBy: null,
    paidBy: null,
    voidedBy: null,
  };

  const pending = {
    ...life,
    id: "dstl_142",
    settlementNumber: "DS-2610-0142",
    status: "PendingApproval",
    classification: "CompanyDriver",
    payProfileName: "Company OTR – mileage bands",
    grossEarningsMinor: 188_680,
    reimbursementsMinor: 12_740,
    deductionsMinor: 33_000,
    carryForwardInMinor: 0,
    carryForwardOutMinor: 0,
    netPayMinor: 168_420,
    totalMiles: "2140.0000",
    shipmentCount: 9,
    submittedAt: anchor - 22 * HOUR,
    createdAt: dayBefore(anchor, 3) + 6 * HOUR,
    updatedAt: anchor - 22 * HOUR,
    exceptions: [
      {
        code: "ManualAdjustment",
        severity: "Warning",
        message: "Layover pay of $150.00 was added by hand, not from a load.",
      },
      {
        code: "HighVariance",
        severity: "Warning",
        message: "Net pay is 41% above the last eight settlements' average of $1,192.00.",
      },
    ],
    worker: { id: "wrk_rosa", firstName: "Rosa", lastName: "Martinez", wholeName: "Rosa Martinez" },
    submittedBy: MIA,
    lineSummary: [
      payGroup({
        componentKind: "Linehaul",
        method: "PerLoadedMile",
        rate: "0.62",
        quantity: "2140",
        amountMinor: 132_680,
        lineCount: 9,
        shipmentCount: 9,
      }),
      payGroup({
        componentKind: "StopPay",
        method: "PerStop",
        rate: "25",
        quantity: "14",
        amountMinor: 35_000,
        lineCount: 9,
        shipmentCount: 9,
      }),
      payGroup({
        componentKind: "Detention",
        method: "PerHour",
        rate: "20",
        quantity: "3",
        amountMinor: 6_000,
        lineCount: 2,
        shipmentCount: 2,
      }),
      payGroup({
        category: "Adjustment",
        description: "Layover - Detroit 10/8",
        amountMinor: 15_000,
        manual: true,
      }),
      payGroup({
        category: "Reimbursement",
        description: "Lumper receipts",
        amountMinor: 8_500,
        lineCount: 2,
        shipmentCount: 2,
      }),
      payGroup({
        category: "Reimbursement",
        description: "Tolls",
        amountMinor: 4_240,
        lineCount: 3,
        shipmentCount: 3,
      }),
      payGroup({
        category: "AdvanceRecovery",
        description: "Advance ADV-0088",
        rate: "200",
        amountMinor: -20_000,
        advanceId: "padv_88",
        advance: {
          id: "padv_88",
          reference: "ADV-0088",
          amountMinor: 50_000,
          outstandingMinor: 30_000,
        },
      }),
      payGroup({
        category: "Deduction",
        description: "Health insurance",
        amountMinor: -9_600,
        recurringDeductionId: "rded_med",
      }),
      payGroup({
        category: "Deduction",
        description: "Phone plan",
        amountMinor: -3_400,
        recurringDeductionId: "rded_phone",
      }),
    ],
    loadShares: [],
    disputes: [
      {
        id: "dsd_1",
        status: "InReview",
        category: "MissingPay",
        description:
          "Three stops on the Memphis run and I was paid for two. The second drop at the Olive Branch DC is missing.",
        resolutionNote: "",
        resolvedAt: null,
        createdAt: anchor - 19 * HOUR,
        settlementLine: { description: "Stop pay", proNumber: "S-240201" },
      },
    ],
  };

  const short = {
    ...pending,
    id: "dstl_147",
    settlementNumber: "DS-2610-0147",
    status: "Draft",
    classification: "OwnerOperator",
    payProfileName: "Lease-on 72% of revenue",
    grossEarningsMinor: 258_720,
    reimbursementsMinor: 0,
    deductionsMinor: 279_960,
    carryForwardOutMinor: -21_240,
    netPayMinor: 0,
    totalMiles: "1180",
    shipmentCount: 3,
    submittedAt: null,
    submittedBy: null,
    worker: { id: "wrk_marcus", firstName: "Marcus", lastName: "Hale", wholeName: "Marcus Hale" },
    exceptions: [
      {
        code: "NegativeNet",
        severity: "Critical",
        message: "Deductions exceeded earnings by $212.40; the balance carries forward.",
      },
    ],
    lineSummary: [
      payGroup({
        componentKind: "Linehaul",
        method: "PercentOfRevenue",
        rate: "72",
        quantity: "3010",
        amountMinor: 216_720,
        lineCount: 3,
        shipmentCount: 3,
      }),
      payGroup({
        componentKind: "FuelSurcharge",
        method: "PercentOfRevenue",
        rate: "100",
        quantity: "420",
        amountMinor: 42_000,
        lineCount: 3,
        shipmentCount: 3,
      }),
      payGroup({
        category: "Deduction",
        description: "Fuel card",
        amountMinor: -269_960,
        lineCount: 6,
      }),
      payGroup({
        category: "EscrowContribution",
        description: "Maintenance escrow",
        rate: "100",
        amountMinor: -10_000,
        escrowAccountId: "escr_1",
        escrowAccount: { id: "escr_1", balanceMinor: 190_000, targetAmountMinor: 250_000 },
      }),
    ],
    disputes: [],
  };

  const paid = {
    ...pending,
    id: "dstl_118",
    settlementNumber: "DS-2610-0118",
    status: "Paid",
    periodStart: dayBefore(anchor, 17),
    periodEnd: dayBefore(anchor, 11),
    payDate: dayBefore(anchor, 5),
    submittedAt: dayBefore(anchor, 9) + 11 * HOUR,
    approvedAt: dayBefore(anchor, 8) + 14 * HOUR,
    postedAt: dayBefore(anchor, 8) + 14 * HOUR,
    paidAt: dayBefore(anchor, 5) + 9 * HOUR,
    paymentMethod: "ACH",
    paymentReference: "0219 4471 2208",
    approvedBy: DANA,
    postedBy: DANA,
    paidBy: DANA,
    exceptions: [],
    disputes: [
      {
        ...pending.disputes[0],
        id: "dsd_0",
        status: "Resolved",
        resolutionNote: "Paid the missing stop on the next settlement.",
        resolvedAt: dayBefore(anchor, 5),
        createdAt: dayBefore(anchor, 7) + 15 * HOUR,
      },
    ],
  };

  const blank = {
    ...pending,
    id: "dstl_150",
    settlementNumber: "DS-2610-0150",
    status: "Draft",
    grossEarningsMinor: 0,
    reimbursementsMinor: 0,
    deductionsMinor: 0,
    netPayMinor: 0,
    totalMiles: "0",
    shipmentCount: 0,
    submittedAt: null,
    submittedBy: null,
    worker: { id: "wrk_jordan", firstName: "Jordan", lastName: "Lee", wholeName: "Jordan Lee" },
    exceptions: [
      {
        code: "NoActivity",
        severity: "Warning",
        message: "No pay events, recurring pay or deductions fell in this period.",
      },
    ],
    lineSummary: [],
    loadShares: [],
    disputes: [],
  };

  const voided = {
    ...pending,
    id: "dstl_131",
    settlementNumber: "DS-2610-0131",
    status: "Voided",
    approvedAt: anchor - 30 * HOUR,
    approvedBy: DANA,
    voidedAt: anchor - 6 * HOUR,
    voidedBy: DANA,
    voidReason: "Generated twice for the same week; DS-2610-0142 stands.",
    exceptions: [],
    disputes: [],
  };

  return {
    [pending.id]: {
      settlement: pending,
      activity: [
        entry("ae_p1", "update", 22, MIA.name, {
          status: { from: "Draft", to: "PendingApproval" },
        }),
        entry("ae_p2", "update", 26, MIA.name, null, "Added layover pay for the Detroit hold"),
        entry("ae_p3", "create", 3 * 24 - 6, null),
      ],
    },
    [short.id]: {
      settlement: short,
      activity: [entry("ae_s1", "create", 3 * 24 - 6, null)],
    },
    [paid.id]: {
      settlement: paid,
      activity: [
        entry("ae_d1", "update", 5 * 24 - 9, DANA.name, { status: { from: "Posted", to: "Paid" } }),
        entry("ae_d2", "update", 8 * 24 - 14, DANA.name, {
          status: { from: "Approved", to: "Posted" },
        }),
        entry("ae_d3", "update", 8 * 24 - 14, DANA.name, {
          status: { from: "PendingApproval", to: "Approved" },
        }),
      ],
    },
    [blank.id]: { settlement: blank, activity: [] },
    [voided.id]: {
      settlement: voided,
      activity: [
        entry("ae_v1", "update", 6, DANA.name, { status: { from: "Approved", to: "Voided" } }),
      ],
    },
  };
}

function cost(overrides) {
  return {
    eventType: "LinehaulCost",
    description: "",
    amountMinor: 0,
    lineCount: 1,
    shipmentCount: 1,
    manual: false,
    ...overrides,
  };
}

function moveInvoice(moveId, proNumber, costMinor, match) {
  return {
    moveId,
    shipmentId: `shp_${proNumber.slice(2)}`,
    proNumber,
    costMinor,
    match:
      match === null
        ? null
        : {
            id: `cim_${moveId}`,
            status: "Matched",
            invoiceNumber: `RBF-${proNumber.slice(2)}`,
            invoiceTotalMinor: costMinor,
            expectedTotalMinor: costMinor,
            varianceMinor: 0,
            currencyCode: "USD",
            duplicateOfMatchId: null,
            ...match,
          },
  };
}

const BLUE_RIDGE = {
  id: "car_ridge",
  code: "RIDGE",
  name: "Blue Ridge Freight LLC",
  mcNumber: "884213",
  paymentMethod: "ACHManual",
  paymentTermDays: 30,
  remitToName: "Blue Ridge Freight Remittance",
  remitAddressLine1: "PO Box 4410",
  remitAddressLine2: "",
  remitCity: "Roanoke",
  remitPostalCode: "24015",
  remitState: { id: "us_va", abbreviation: "VA" },
};

/** The carrier settlements, keyed by ID, each with its activity. */
export function deskCarrierSettlements(anchor) {
  const entry = entryAt(anchor);
  const start = dayBefore(anchor, 10);
  const end = dayBefore(anchor, 4);
  const life = {
    periodStart: start,
    periodEnd: end,
    payDate: dayBefore(anchor, -26),
    currencyCode: "USD",
    approvedAt: null,
    postedAt: null,
    paidAt: null,
    paymentMethod: "",
    paymentReference: "",
    voidedAt: null,
    voidReason: "",
    approvedBy: null,
    postedBy: null,
    paidBy: null,
    voidedBy: null,
  };

  const pending = {
    ...life,
    id: "cstl_88",
    settlementNumber: "CS-2610-0088",
    status: "PendingApproval",
    grossCostMinor: 1_062_500,
    adjustmentsMinor: -2_500,
    netPayableMinor: 1_060_000,
    shipmentCount: 6,
    submittedAt: anchor - 20 * HOUR,
    createdAt: dayBefore(anchor, 3) + 6 * HOUR,
    updatedAt: anchor - 20 * HOUR,
    carrier: BLUE_RIDGE,
    submittedBy: MIA,
    costSummary: [
      cost({
        description: "Rate confirmation",
        amountMinor: 960_000,
        lineCount: 6,
        shipmentCount: 6,
      }),
      cost({ eventType: "FuelSurcharge", amountMinor: 87_500, lineCount: 6, shipmentCount: 6 }),
      cost({ eventType: "Accessorial", description: "Lumper", amountMinor: 15_000 }),
      cost({
        eventType: "Adjustment",
        description: "Invoice variance RBF-240302",
        amountMinor: 6_000,
      }),
      cost({
        eventType: "Adjustment",
        description: "Late delivery credit",
        amountMinor: -8_500,
        shipmentCount: 0,
        manual: true,
      }),
    ],
    loadShares: [],
    moveInvoices: [
      moveInvoice("smv_301", "S-240301", 184_500, {}),
      moveInvoice("smv_302", "S-240302", 171_000, { status: "Resolved", varianceMinor: 6_000 }),
      moveInvoice("smv_303", "S-240303", 175_000, {
        status: "Variance",
        invoiceTotalMinor: 183_500,
        expectedTotalMinor: 175_000,
        varianceMinor: 8_500,
      }),
      moveInvoice("smv_304", "S-240304", 178_000, {}),
      moveInvoice("smv_305", "S-240305", 179_500, {}),
      moveInvoice("smv_306", "S-240306", 172_000, null),
    ],
  };

  const approved = {
    ...pending,
    id: "cstl_90",
    settlementNumber: "CS-2610-0090",
    status: "Approved",
    approvedAt: anchor - 3 * HOUR,
    approvedBy: DANA,
    grossCostMinor: 512_000,
    adjustmentsMinor: 0,
    netPayableMinor: 512_000,
    shipmentCount: 3,
    carrier: {
      ...BLUE_RIDGE,
      id: "car_lonestar",
      code: "LSH",
      name: "Lone Star Haulers",
      mcNumber: "612077",
      paymentMethod: "Check",
      paymentTermDays: 15,
      remitToName: "",
      remitAddressLine1: "2210 Industrial Blvd",
      remitCity: "Fort Worth",
      remitPostalCode: "76106",
      remitState: { id: "us_tx", abbreviation: "TX" },
    },
    costSummary: [
      cost({
        description: "Rate confirmation",
        amountMinor: 468_000,
        lineCount: 3,
        shipmentCount: 3,
      }),
      cost({ eventType: "FuelSurcharge", amountMinor: 44_000, lineCount: 3, shipmentCount: 3 }),
    ],
    moveInvoices: [
      moveInvoice("smv_411", "S-240411", 171_000, {
        status: "Suggested",
        invoiceNumber: "LSH-9917",
      }),
      moveInvoice("smv_412", "S-240412", 166_000, {
        status: "Variance",
        invoiceNumber: "LSH-9918",
        invoiceTotalMinor: 158_500,
        expectedTotalMinor: 166_000,
        varianceMinor: -7_500,
      }),
      moveInvoice("smv_413", "S-240413", 175_000, { invoiceNumber: "LSH-9920" }),
    ],
  };

  const paid = {
    ...pending,
    id: "cstl_71",
    settlementNumber: "CS-2609-0071",
    status: "Paid",
    periodStart: dayBefore(anchor, 45),
    periodEnd: dayBefore(anchor, 39),
    payDate: dayBefore(anchor, 8),
    submittedAt: dayBefore(anchor, 37) + 10 * HOUR,
    approvedAt: dayBefore(anchor, 36) + 15 * HOUR,
    postedAt: dayBefore(anchor, 36) + 16 * HOUR,
    paidAt: dayBefore(anchor, 8) + 11 * HOUR,
    paymentMethod: "Check",
    paymentReference: "CHK 10442",
    approvedBy: DANA,
    postedBy: DANA,
    paidBy: LI,
    moveInvoices: pending.moveInvoices.map((move) =>
      move.match ? { ...move, match: { ...move.match, status: "Resolved" } } : move,
    ),
  };

  const blank = {
    ...pending,
    id: "cstl_91",
    settlementNumber: "CS-2610-0091",
    status: "Draft",
    grossCostMinor: 0,
    adjustmentsMinor: 0,
    netPayableMinor: 0,
    shipmentCount: 0,
    submittedAt: null,
    submittedBy: null,
    carrier: {
      ...BLUE_RIDGE,
      id: "car_mesa",
      code: "MESA",
      name: "Mesa Haulers",
      mcNumber: "",
      paymentMethod: "Check",
      paymentTermDays: 0,
      remitToName: "",
      remitAddressLine1: "",
      remitAddressLine2: "",
      remitCity: "",
      remitPostalCode: "",
      remitState: null,
    },
    costSummary: [],
    loadShares: [],
    moveInvoices: [],
  };

  const voided = {
    ...pending,
    id: "cstl_87",
    settlementNumber: "CS-2610-0087",
    status: "Voided",
    approvedAt: anchor - 28 * HOUR,
    approvedBy: DANA,
    voidedAt: anchor - 5 * HOUR,
    voidedBy: DANA,
    voidReason: "Duplicate of CS-2610-0088; the batch ran twice for the same week.",
  };

  return {
    [pending.id]: {
      settlement: pending,
      activity: [
        entry("ae_c1", "update", 20, MIA.name, {
          status: { from: "Draft", to: "PendingApproval" },
        }),
        entry("ae_c2", "update", 23, MIA.name, null, "Accepted RBF-240302's $60.00 detention"),
        entry("ae_c3", "create", 3 * 24 - 6, null),
      ],
    },
    [approved.id]: {
      settlement: approved,
      activity: [
        entry("ae_a1", "update", 3, DANA.name, {
          status: { from: "PendingApproval", to: "Approved" },
        }),
      ],
    },
    [paid.id]: {
      settlement: paid,
      activity: [
        entry("ae_k1", "update", 8 * 24 - 11, LI.name, { status: { from: "Posted", to: "Paid" } }),
      ],
    },
    [blank.id]: { settlement: blank, activity: [] },
    [voided.id]: {
      settlement: voided,
      activity: [
        entry("ae_x1", "update", 5, DANA.name, { status: { from: "Approved", to: "Voided" } }),
      ],
    },
  };
}

/** The cards thread's worker settlement conversations. */
export const DESK_DRIVER_SETTLEMENT_CONVERSATIONS = {
  "pay-rosa": {
    recordId: "dstl_142",
    title: "Settlement DS-2610-0142",
    question: "Can I approve Rosa's settlement for last week?",
    reply:
      "It nets $1,684.20 on 9 loads and is waiting on you. Two things to look at first: layover pay was added by hand, and Rosa has a dispute open about a missing stop. [DS-2610-0142](artifact:art_pay-rosa) has the flags and the dispute.",
  },
  "pay-marcus": {
    recordId: "dstl_147",
    title: "Settlement DS-2610-0147",
    question: "Why is Marcus getting nothing this week?",
    reply:
      "His fuel card ran past what his three loads earned, so the settlement pays $0.00 and carries $212.40 to the next one. [DS-2610-0147](artifact:art_pay-marcus) shows how it adds up.",
  },
  "pay-paid": {
    recordId: "dstl_118",
    title: "Settlement DS-2610-0118",
    question: "Did Rosa's previous settlement go out?",
    reply:
      "Yes, Dana paid it by ACH on its pay date. [DS-2610-0118](artifact:art_pay-paid) has who signed each step.",
  },
  "pay-jordan": {
    recordId: "dstl_150",
    title: "Settlement DS-2610-0150",
    question: "What's on Jordan's settlement?",
    reply:
      "Nothing yet: no loads, recurring pay or deductions fell in the period. [DS-2610-0150](artifact:art_pay-jordan) is the draft.",
  },
  "pay-voided": {
    recordId: "dstl_131",
    title: "Settlement DS-2610-0131",
    question: "What happened to DS-2610-0131?",
    reply:
      "Dana voided it this morning; the week had been generated twice. [DS-2610-0131](artifact:art_pay-voided) keeps why.",
  },
  "pay-gone": {
    recordId: "dstl_099",
    title: "Settlement DS-2609-0099",
    question: "Pull up DS-2609-0099.",
    reply:
      "That settlement has been removed since I looked it up. [DS-2609-0099](artifact:art_pay-gone) is what I found then.",
  },
};

/** The cards thread's carrier settlement conversations. */
export const DESK_CARRIER_SETTLEMENT_CONVERSATIONS = {
  "payable-ridge": {
    recordId: "cstl_88",
    title: "Settlement CS-2610-0088",
    question: "Is Blue Ridge's payable ready to approve?",
    reply:
      "Nearly: four of its six invoices match, one is $85.00 over the buy rate and one load hasn't been invoiced. [CS-2610-0088](artifact:art_payable-ridge) lists them.",
  },
  "payable-lonestar": {
    recordId: "cstl_90",
    title: "Settlement CS-2610-0090",
    question: "Can we post Lone Star's settlement?",
    reply:
      "It's approved and ready to post, though one invoice is still waiting to be accepted and another came in $75.00 under. [CS-2610-0090](artifact:art_payable-lonestar) has both.",
  },
  "payable-paid": {
    recordId: "cstl_71",
    title: "Settlement CS-2609-0071",
    question: "Did we pay Blue Ridge for September?",
    reply:
      "Yes, Li mailed check CHK 10442 on its pay date. [CS-2609-0071](artifact:art_payable-paid) has the sign-offs.",
  },
  "payable-mesa": {
    recordId: "cstl_91",
    title: "Settlement CS-2610-0091",
    question: "What do we owe Mesa Haulers?",
    reply:
      "Nothing yet; none of their loads has delivered, and they have no remit-to address on file. [CS-2610-0091](artifact:art_payable-mesa) is the draft.",
  },
  "payable-voided": {
    recordId: "cstl_87",
    title: "Settlement CS-2610-0087",
    question: "Why was CS-2610-0087 voided?",
    reply:
      "The batch ran twice for the same week, so Dana voided the duplicate. [CS-2610-0087](artifact:art_payable-voided) keeps why.",
  },
  "payable-gone": {
    recordId: "cstl_050",
    title: "Settlement CS-2608-0050",
    question: "Pull up CS-2608-0050.",
    reply:
      "That settlement has been removed since I looked it up. [CS-2608-0050](artifact:art_payable-gone) is what I found then.",
  },
};
