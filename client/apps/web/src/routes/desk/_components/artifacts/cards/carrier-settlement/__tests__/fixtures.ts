import type {
  DeskCarrierMoveInvoice,
  DeskCarrierSettlement,
  DeskCarrierSettlementGroup,
} from "@/lib/graphql/settlement-card";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/** Midnight UTC of a day in 2026, as the server's dates read. */
export const DAY = {
  oct5: 1_791_158_400,
  oct9: 1_791_504_000,
  oct11: 1_791_676_800,
  oct12: 1_791_763_200,
  oct13: 1_791_849_600,
  oct14: 1_791_936_000,
  nov10: 1_794_268_800,
} as const;

export const HOUR = 3_600;

/** A translator that fills ICU arguments and plurals the way the catalog does. */
export const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

/** A row of the costs as the server sums them: one kind and description, its lines and loads. */
export function cost(
  overrides: Partial<DeskCarrierSettlementGroup> = {},
): DeskCarrierSettlementGroup {
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

type Match = NonNullable<DeskCarrierMoveInvoice["match"]>;

/** A move on the statement with the carrier's standing invoice for it, or none. */
export function move(
  overrides: Partial<Omit<DeskCarrierMoveInvoice, "match">> & {
    match?: Partial<Match> | null;
  } = {},
): DeskCarrierMoveInvoice {
  const { match, ...rest } = overrides;
  const base = {
    moveId: "smv_1",
    shipmentId: "shp_1",
    proNumber: "S-240301",
    costMinor: 180_000,
    ...rest,
  };
  return {
    ...base,
    match:
      match === null
        ? null
        : {
            id: `cim_${base.moveId}`,
            status: "Matched",
            invoiceNumber: `INV-${base.proNumber}`,
            invoiceTotalMinor: base.costMinor,
            expectedTotalMinor: base.costMinor,
            varianceMinor: 0,
            currencyCode: "USD",
            duplicateOfMatchId: null,
            ...match,
          },
  };
}

/**
 * A carrier's weekly payable waiting on approval: six moves, four matched,
 * one invoiced $85.00 over the buy rate, one not invoiced yet; linehaul, fuel
 * and a lumper over an accepted variance and a credit added by hand.
 */
export function pendingSettlement(
  overrides: Partial<DeskCarrierSettlement> = {},
): DeskCarrierSettlement {
  return {
    id: "cstl_88",
    settlementNumber: "CS-2610-0088",
    status: "PendingApproval",
    periodStart: DAY.oct5,
    periodEnd: DAY.oct11,
    payDate: DAY.nov10,
    grossCostMinor: 1_062_500,
    adjustmentsMinor: -2_500,
    netPayableMinor: 1_060_000,
    shipmentCount: 6,
    currencyCode: "USD",
    submittedAt: DAY.oct13 + 15 * HOUR,
    approvedAt: null,
    postedAt: null,
    paidAt: null,
    paymentMethod: "",
    paymentReference: "",
    voidedAt: null,
    voidReason: "",
    createdAt: DAY.oct12 + 6 * HOUR,
    updatedAt: DAY.oct13 + 15 * HOUR,
    carrier: {
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
    },
    submittedBy: { id: "usr_mia", name: "Mia Chen" },
    approvedBy: null,
    postedBy: null,
    paidBy: null,
    voidedBy: null,
    costSummary: [
      cost({
        eventType: "LinehaulCost",
        description: "Rate confirmation",
        amountMinor: 960_000,
        lineCount: 6,
        shipmentCount: 6,
      }),
      cost({ eventType: "FuelSurcharge", amountMinor: 87_500, lineCount: 6, shipmentCount: 6 }),
      cost({
        eventType: "Accessorial",
        description: "Lumper",
        amountMinor: 15_000,
        lineCount: 1,
        shipmentCount: 1,
      }),
      cost({
        eventType: "Adjustment",
        description: "Invoice variance INV-77120",
        amountMinor: 6_000,
        lineCount: 1,
        shipmentCount: 1,
      }),
      cost({
        eventType: "Adjustment",
        description: "Late delivery credit",
        amountMinor: -8_500,
        lineCount: 1,
        shipmentCount: 0,
        manual: true,
      }),
    ],
    loadShares: [
      { shipmentId: "shp_240301", proNumber: "S-240301", amountMinor: 184_500 },
      { shipmentId: "shp_240302", proNumber: "S-240302", amountMinor: 171_000 },
    ],
    moveInvoices: [
      move({ moveId: "smv_1", shipmentId: "shp_240301", proNumber: "S-240301" }),
      move({
        moveId: "smv_2",
        shipmentId: "shp_240302",
        proNumber: "S-240302",
        match: { status: "Resolved" },
      }),
      move({
        moveId: "smv_3",
        shipmentId: "shp_240303",
        proNumber: "S-240303",
        costMinor: 175_000,
        match: {
          status: "Variance",
          invoiceNumber: "INV-77141",
          invoiceTotalMinor: 183_500,
          expectedTotalMinor: 175_000,
          varianceMinor: 8_500,
        },
      }),
      move({ moveId: "smv_4", shipmentId: "shp_240304", proNumber: "S-240304" }),
      move({ moveId: "smv_5", shipmentId: "shp_240305", proNumber: "S-240305" }),
      move({
        moveId: "smv_6",
        shipmentId: "shp_240306",
        proNumber: "S-240306",
        costMinor: 172_000,
        match: null,
      }),
    ],
    ...overrides,
  };
}

/** The same payable paid by check after it was posted, with every step signed. */
export function paidSettlement(
  overrides: Partial<DeskCarrierSettlement> = {},
): DeskCarrierSettlement {
  return pendingSettlement({
    status: "Paid",
    approvedAt: DAY.oct13 + 18 * HOUR,
    postedAt: DAY.oct14 + 9 * HOUR,
    paidAt: DAY.nov10,
    paymentMethod: "Check",
    paymentReference: "CHK 10442",
    approvedBy: { id: "usr_dana", name: "Dana Ortiz" },
    postedBy: { id: "usr_dana", name: "Dana Ortiz" },
    paidBy: { id: "usr_li", name: "Li Wei" },
    ...overrides,
  });
}

/** The payable voided before it was posted, generated twice for the same week. */
export function voidedSettlement(
  overrides: Partial<DeskCarrierSettlement> = {},
): DeskCarrierSettlement {
  return pendingSettlement({
    status: "Voided",
    approvedAt: DAY.oct13 + 18 * HOUR,
    voidedAt: DAY.oct14 + 10 * HOUR,
    voidReason: "Duplicate of CS-2610-0087.",
    approvedBy: { id: "usr_dana", name: "Dana Ortiz" },
    voidedBy: { id: "usr_dana", name: "Dana Ortiz" },
    ...overrides,
  });
}

/**
 * A draft for a carrier with nothing delivered yet: no costs, no loads, no
 * remit-to address and no payment terms; the server read neither summary.
 */
export function emptySettlement(
  overrides: Partial<DeskCarrierSettlement> = {},
): DeskCarrierSettlement {
  return pendingSettlement({
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
    costSummary: null,
    loadShares: null,
    moveInvoices: null,
    ...overrides,
  });
}
