import type { DeskIftaReturnCard, DeskIftaReturnLine } from "@/lib/graphql/ifta-return-desk-card";

/** Nov 23, 2026, 9:41 AM UTC: after the third quarter's filing date, inside the fourth quarter. */
export const NOW = Date.UTC(2026, 10, 23, 9, 41) / 1000;
const Q3_END = Date.UTC(2026, 9, 1) / 1000;
const Q3_DUE = Date.UTC(2026, 9, 31) / 1000;
const Q4_END = Date.UTC(2027, 0, 1) / 1000;
const Q4_DUE = Date.UTC(2027, 0, 31) / 1000;

export function line(overrides: Partial<DeskIftaReturnLine> = {}): DeskIftaReturnLine {
  return {
    id: "ifrl_georgia",
    fuelType: "Diesel",
    isIftaMember: true,
    totalMiles: "14300",
    taxableMiles: "14300",
    taxPaidGallons: "456",
    netTaxableGallons: "1856",
    ratePerGallon: "0.3288",
    rateMissing: false,
    taxDue: "610.32",
    surchargeDue: "0",
    lineTotal: "610.32",
    sortOrder: 3,
    jurisdiction: { id: "ifj_ga", code: "GA", name: "Georgia" },
    ...overrides,
  };
}

/** The third quarter as a draft past its filing date: Oklahoma unpriced, Texas the largest credit. */
export function draftCard(overrides: Partial<DeskIftaReturnCard> = {}): DeskIftaReturnCard {
  return {
    id: "ifr_q3",
    year: 2026,
    quarter: 3,
    period: { label: "Q3 2026", end: Q3_END, dueDate: Q3_DUE },
    amendmentNumber: 0,
    status: "Draft",
    totalMiles: "186420",
    totalTaxableMiles: "186420",
    totalGallons: "28608",
    totalTaxPaidGallons: "28608",
    netTaxableGallons: "1204",
    taxDue: "258.26",
    surchargeDue: "0",
    netDue: "258.26",
    currencyCode: "USD",
    unattributedMiles: "412",
    unattributedMoveCount: 3,
    noTractorMiles: "0",
    noTractorMoveCount: 0,
    problems: [
      {
        code: "MissingRate",
        message: "No Diesel tax rate is on file for Oklahoma in Q3 2026",
        jurisdictionCode: "US_OK",
        fuelType: "Diesel",
        amount: null,
      },
      {
        code: "UnattributedMiles",
        message: "3 completed moves totalling 412.00 miles have no jurisdiction breakdown",
        jurisdictionCode: null,
        fuelType: null,
        amount: "412.00",
      },
      {
        code: "NonMemberActivity",
        message: "Mexico is not an IFTA member",
        jurisdictionCode: "MX_NL",
        fuelType: "Diesel",
        amount: "120.00",
      },
      {
        code: "NonMemberActivity",
        message: "Mexico is not an IFTA member",
        jurisdictionCode: "MX_CH",
        fuelType: "Diesel",
        amount: "80.00",
      },
    ],
    computedAt: NOW - 3600,
    finalizedAt: null,
    filedAt: null,
    filingReference: null,
    lines: [
      line(),
      line({
        id: "ifrl_texas",
        totalMiles: "48210",
        taxPaidGallons: "14160",
        lineTotal: "-1352.40",
        taxDue: "-1352.40",
        jurisdiction: { id: "ifj_tx", code: "TX", name: "Texas" },
      }),
      line({
        id: "ifrl_oklahoma",
        totalMiles: "21340",
        taxPaidGallons: "2532",
        rateMissing: true,
        ratePerGallon: null,
        lineTotal: "0",
        taxDue: "0",
        jurisdiction: { id: "ifj_ok", code: "OK", name: "Oklahoma" },
      }),
      line({
        id: "ifrl_arkansas",
        lineTotal: "464.78",
        jurisdiction: { id: "ifj_ar", code: "AR", name: "Arkansas" },
      }),
    ],
    ...overrides,
  };
}

/** The fourth quarter, still running and clear of problems. */
export function openQuarterCard(overrides: Partial<DeskIftaReturnCard> = {}): DeskIftaReturnCard {
  return draftCard({
    id: "ifr_q4",
    quarter: 4,
    period: { label: "Q4 2026", end: Q4_END, dueDate: Q4_DUE },
    problems: [],
    netDue: "-42.10",
    ...overrides,
  });
}

/** Ten priced lines, so the card counts what it does not draw. */
export function manyLines(): DeskIftaReturnLine[] {
  return Array.from({ length: 10 }, (_, index) =>
    line({
      id: `ifrl_${index}`,
      lineTotal: String(100 + index),
      jurisdiction: { id: `ifj_${index}`, code: `J${index}`, name: `Jurisdiction ${index}` },
    }),
  );
}
