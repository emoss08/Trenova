import type {
  DeskDriverSettlement,
  DeskDriverSettlementDispute,
  DeskDriverSettlementGroup,
} from "@/lib/graphql/settlement-card";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/** Midnight UTC of a day in 2026, as the server's dates read. */
export const DAY = {
  sep28: 1_790_553_600,
  oct4: 1_791_072_000,
  oct5: 1_791_158_400,
  oct6: 1_791_244_800,
  oct7: 1_791_331_200,
  oct10: 1_791_590_400,
  oct11: 1_791_676_800,
  oct12: 1_791_763_200,
  oct13: 1_791_849_600,
  oct14: 1_791_936_000,
  oct17: 1_792_195_200,
} as const;

export const HOUR = 3_600;

/** A translator that fills ICU arguments and plurals the way the catalog does. */
export const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

/** A row of the statement as the server sums it: one way of paying, its lines and loads. */
export function group(
  overrides: Partial<DeskDriverSettlementGroup> = {},
): DeskDriverSettlementGroup {
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

export function dispute(
  overrides: Partial<DeskDriverSettlementDispute> = {},
): DeskDriverSettlementDispute {
  return {
    id: "dsd_1",
    status: "InReview",
    category: "MissingPay",
    description:
      "Three stops on the Memphis run and I was paid for two. The second drop at the Olive Branch DC is missing.",
    resolutionNote: "",
    resolvedAt: null,
    createdAt: DAY.oct12 + 19 * HOUR,
    settlementLine: { description: "Stop pay", proNumber: "S-240201" },
    ...overrides,
  };
}

/**
 * A company worker's weekly settlement waiting on approval: nine loads, stop
 * pay and detention, layover pay added by hand, lumper and toll
 * reimbursements, an advance being recovered and two recurring deductions,
 * with two flags and a dispute open.
 */
export function pendingSettlement(
  overrides: Partial<DeskDriverSettlement> = {},
): DeskDriverSettlement {
  return {
    id: "dstl_142",
    settlementNumber: "DS-2610-0142",
    status: "PendingApproval",
    classification: "CompanyDriver",
    payProfileName: "Company OTR – mileage bands",
    periodStart: DAY.oct5,
    periodEnd: DAY.oct11,
    payDate: DAY.oct17,
    grossEarningsMinor: 188_680,
    reimbursementsMinor: 12_740,
    deductionsMinor: 33_000,
    carryForwardInMinor: 0,
    carryForwardOutMinor: 0,
    netPayMinor: 168_420,
    totalMiles: "2140.0000",
    shipmentCount: 9,
    currencyCode: "USD",
    submittedAt: DAY.oct13 + 16 * HOUR,
    approvedAt: null,
    postedAt: null,
    paidAt: null,
    paymentMethod: "",
    paymentReference: "",
    voidedAt: null,
    voidReason: "",
    createdAt: DAY.oct12 + 6 * HOUR,
    updatedAt: DAY.oct13 + 16 * HOUR,
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
    submittedBy: { id: "usr_mia", name: "Mia Chen" },
    approvedBy: null,
    postedBy: null,
    paidBy: null,
    voidedBy: null,
    lineSummary: [
      group({
        componentKind: "Linehaul",
        method: "PerLoadedMile",
        description: "",
        rate: "0.62",
        quantity: "2140",
        amountMinor: 132_680,
        lineCount: 9,
        shipmentCount: 9,
      }),
      group({
        componentKind: "StopPay",
        method: "PerStop",
        rate: "25",
        quantity: "14",
        amountMinor: 35_000,
        lineCount: 9,
        shipmentCount: 9,
      }),
      group({
        componentKind: "Detention",
        method: "PerHour",
        rate: "20",
        quantity: "3",
        amountMinor: 6_000,
        lineCount: 2,
        shipmentCount: 2,
      }),
      group({
        category: "Adjustment",
        description: "Layover - Detroit 10/8",
        amountMinor: 15_000,
        manual: true,
      }),
      group({
        category: "Reimbursement",
        description: "Lumper receipts",
        amountMinor: 8_500,
        lineCount: 2,
        shipmentCount: 2,
      }),
      group({
        category: "Reimbursement",
        description: "Tolls",
        amountMinor: 4_240,
        lineCount: 3,
        shipmentCount: 3,
      }),
      group({
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
      group({
        category: "Deduction",
        description: "Health insurance",
        amountMinor: -9_600,
        recurringDeductionId: "rded_med",
      }),
      group({
        category: "Deduction",
        description: "Phone plan",
        amountMinor: -3_400,
        recurringDeductionId: "rded_phone",
      }),
    ],
    loadShares: [
      { shipmentId: "shp_240193", proNumber: "S-240193", amountMinor: 61_220 },
      { shipmentId: "shp_240201", proNumber: "S-240201", amountMinor: 34_990 },
    ],
    disputes: [dispute()],
    ...overrides,
  };
}

/** An owner-operator whose deductions ran past earnings: nothing paid, the shortfall carried forward. */
export function shortSettlement(
  overrides: Partial<DeskDriverSettlement> = {},
): DeskDriverSettlement {
  return pendingSettlement({
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
    exceptions: [
      {
        code: "NegativeNet",
        severity: "Critical",
        message: "Deductions exceeded earnings by $212.40; the balance carries forward.",
      },
    ],
    lineSummary: [
      group({
        componentKind: "Linehaul",
        method: "PercentOfRevenue",
        rate: "72",
        quantity: "3010",
        amountMinor: 216_720,
        lineCount: 3,
        shipmentCount: 3,
      }),
      group({
        componentKind: "FuelSurcharge",
        method: "PercentOfRevenue",
        rate: "100",
        quantity: "420",
        amountMinor: 42_000,
        lineCount: 3,
        shipmentCount: 3,
      }),
      group({
        category: "Deduction",
        description: "Fuel card",
        amountMinor: -269_960,
        lineCount: 6,
        recurringDeductionId: null,
      }),
      group({
        category: "EscrowContribution",
        description: "Maintenance escrow",
        rate: "100",
        amountMinor: -10_000,
        escrowAccountId: "escr_1",
        escrowAccount: { id: "escr_1", balanceMinor: 190_000, targetAmountMinor: 250_000 },
      }),
    ],
    disputes: [],
    ...overrides,
  });
}

/** A settlement paid by ACH on its pay date, every step signed. */
export function paidSettlement(
  overrides: Partial<DeskDriverSettlement> = {},
): DeskDriverSettlement {
  return pendingSettlement({
    id: "dstl_118",
    settlementNumber: "DS-2610-0118",
    status: "Paid",
    periodStart: DAY.sep28,
    periodEnd: DAY.oct4,
    payDate: DAY.oct10,
    submittedAt: DAY.oct6 + 11 * HOUR,
    approvedAt: DAY.oct7 + 14 * HOUR,
    postedAt: DAY.oct7 + 14 * HOUR,
    paidAt: DAY.oct10 + 9 * HOUR,
    paymentMethod: "ACH",
    paymentReference: "0219 4471 2208",
    approvedBy: { id: "usr_dana", name: "Dana Ruiz" },
    postedBy: { id: "usr_dana", name: "Dana Ruiz" },
    paidBy: { id: "usr_dana", name: "Dana Ruiz" },
    disputes: [
      dispute({
        id: "dsd_0",
        status: "Resolved",
        resolutionNote: "Paid the missing stop on the next settlement.",
        resolvedAt: DAY.oct10,
      }),
    ],
    ...overrides,
  });
}

/** A draft with nothing on it: no loads, no recurring pay, no deductions. */
export function emptySettlement(
  overrides: Partial<DeskDriverSettlement> = {},
): DeskDriverSettlement {
  return pendingSettlement({
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
    ...overrides,
  });
}

/** A settlement voided after it was approved, before it posted. */
export function voidedSettlement(
  overrides: Partial<DeskDriverSettlement> = {},
): DeskDriverSettlement {
  return pendingSettlement({
    id: "dstl_131",
    settlementNumber: "DS-2610-0131",
    status: "Voided",
    approvedAt: DAY.oct13,
    approvedBy: { id: "usr_dana", name: "Dana Ruiz" },
    voidedAt: DAY.oct14 + 9 * HOUR,
    voidedBy: { id: "usr_dana", name: "Dana Ruiz" },
    voidReason: "Generated twice for the same week; DS-2610-0142 stands.",
    ...overrides,
  });
}
