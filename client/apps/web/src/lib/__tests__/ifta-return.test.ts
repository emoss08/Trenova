import type { IftaReturnLineFieldsFragment } from "@trenova/graphql/generated/graphql";
import { IFTA_MIN_YEAR } from "@trenova/shared/types/ifta-tax-rate";
import { describe, expect, it } from "vitest";
import {
  canFinalize,
  defaultIftaPeriod,
  formatIftaMeasure,
  formatIftaMoney,
  iftaYearOptions,
  groupLinesByFuelType,
  IFTA_RETURN_CSV_COLUMNS,
  iftaReturnCsvFilename,
  iftaReturnCsvRows,
  isCreditLine,
  linesMissingRates,
  netPosition,
  periodFromSearch,
  periodInclusiveEnd,
  problemTone,
  problemTotals,
  quarterLabel,
  returnTransitions,
  type IftaReturnView,
  sumLines,
  type IftaReturnPermissions,
} from "../ifta-return";

const UTC = "UTC";

// 2026-01-01T00:00:00Z, 2026-04-01T00:00:00Z, 2026-12-31T23:59:59Z
const JAN_1_2026 = 1_767_225_600;
const APR_1_2026 = 1_775_001_600;
const DEC_31_2026 = 1_798_761_599;

function line(over: Partial<IftaReturnLineFieldsFragment> = {}): IftaReturnLineFieldsFragment {
  return {
    id: over.id ?? "irl_1",
    returnId: "ir_1",
    jurisdictionId: "ij_tx",
    fuelType: "Diesel",
    isIftaMember: true,
    totalMiles: "1000.00",
    taxableMiles: "1000.00",
    routeMiles: "900.00",
    manualMiles: "100.00",
    loadedMiles: "800.00",
    emptyMiles: "200.00",
    taxPaidGallons: "150",
    taxPaidGallonsRaw: "150.250",
    purchaseCount: 3,
    taxableGallons: "160",
    netTaxableGallons: "10",
    ratePerGallon: "0.2000",
    surchargeRatePerGallon: null,
    rateMissing: false,
    taxDue: "2.00",
    surchargeDue: "0.00",
    lineTotal: "2.00",
    sortOrder: 1,
    jurisdiction: {
      id: "ij_tx",
      countryCode: "US",
      code: "TX",
      name: "Texas",
      isIftaMember: true,
      hasSurcharge: false,
    },
    ...over,
  };
}

function ret(over: Partial<IftaReturnView> = {}): IftaReturnView {
  return {
    id: "ir_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    year: 2026,
    quarter: 2,
    period: {
      year: 2026,
      quarter: 2,
      key: "2026Q2",
      label: "Q2 2026",
      start: APR_1_2026,
      end: 1_782_864_000,
      dueDate: 1_785_456_000,
    },
    amendmentNumber: 0,
    amendsReturnId: null,
    amendsReturn: null,
    status: "Draft",
    timezone: "America/New_York",
    periodStart: APR_1_2026,
    periodEnd: 1_782_864_000,
    totalMiles: "1000.00",
    totalTaxableMiles: "1000.00",
    totalGallons: "160.000",
    totalTaxPaidGallons: "150",
    netTaxableGallons: "10",
    taxDue: "2.00",
    surchargeDue: "0.00",
    netDue: "2.00",
    currencyCode: "USD",
    fleetMpgByFuelType: [
      { fuelType: "Diesel", mpg: "6.25", totalMiles: "1000.00", totalGallons: "160.000" },
    ],
    unattributedMiles: "0.00",
    unattributedMoveCount: 0,
    noTractorMiles: "0.00",
    noTractorMoveCount: 0,
    problems: [],
    computedAt: 1_783_000_000,
    finalizedAt: null,
    finalizedById: null,
    finalizedBy: null,
    filedAt: null,
    filedById: null,
    filedBy: null,
    filingReference: null,
    reopenedAt: null,
    reopenedById: null,
    reopenReason: null,
    lines: [line()],
    version: 1,
    createdAt: 1,
    updatedAt: 2,
    ...over,
  };
}

const NO_PERMS: IftaReturnPermissions = {
  create: false,
  update: false,
  approve: false,
  reopen: false,
  submit: false,
  delete: false,
  export: false,
};

const ALL_PERMS: IftaReturnPermissions = {
  create: true,
  update: true,
  approve: true,
  reopen: true,
  submit: true,
  delete: true,
  export: true,
};

describe("defaultIftaPeriod", () => {
  // The page opens on the quarter a preparer is filing, which is the one that
  // has just closed, never the one still running.
  it("is the previous year's Q4 on New Year's Day", () => {
    expect(defaultIftaPeriod(JAN_1_2026, UTC)).toEqual({ year: 2025, quarter: 4 });
  });

  it("is Q1 on the first day of Q2", () => {
    expect(defaultIftaPeriod(APR_1_2026, UTC)).toEqual({ year: 2026, quarter: 1 });
  });

  it("is Q3 on the last day of the year", () => {
    expect(defaultIftaPeriod(DEC_31_2026, UTC)).toEqual({ year: 2026, quarter: 3 });
  });
});

describe("quarterLabel and periodFromSearch", () => {
  it("labels a period the way the return does", () => {
    expect(quarterLabel({ year: 2026, quarter: 2 })).toBe("Q2 2026");
  });

  it("reads a valid year and quarter from the URL", () => {
    expect(periodFromSearch("2025", "3", { year: 2026, quarter: 1 })).toEqual({
      year: 2025,
      quarter: 3,
    });
  });

  it("falls back for anything the URL gets wrong", () => {
    const fallback = { year: 2026, quarter: 1 };
    expect(periodFromSearch(null, null, fallback)).toEqual(fallback);
    expect(periodFromSearch("abc", "2", fallback)).toEqual(fallback);
    expect(periodFromSearch("2026", "5", fallback)).toEqual(fallback);
    expect(periodFromSearch("2026", "0", fallback)).toEqual(fallback);
    expect(periodFromSearch("1999", "2", fallback)).toEqual(fallback);
    expect(periodFromSearch("2026.5", "2", fallback)).toEqual(fallback);
  });
});

describe("sumLines", () => {
  it("adds decimal strings without float drift and keeps credit rows signed", () => {
    const totals = sumLines([
      line({ id: "a", taxDue: "0.10", lineTotal: "0.10", netTaxableGallons: "1" }),
      line({ id: "b", taxDue: "0.20", lineTotal: "0.20", netTaxableGallons: "2" }),
      line({ id: "c", taxDue: "-0.30", lineTotal: "-0.30", netTaxableGallons: "-3" }),
      line({ id: "d", taxDue: "0.00", lineTotal: "0.00", netTaxableGallons: "0" }),
    ]);
    expect(totals.taxDue).toBe("0.00");
    expect(totals.lineTotal).toBe("0.00");
    expect(totals.netTaxableGallons).toBe("0");
    expect(totals.totalMiles).toBe("4000.00");
    expect(totals.taxPaidGallons).toBe("600");
  });

  it("returns zeros for no lines", () => {
    const totals = sumLines([]);
    expect(totals.totalMiles).toBe("0.00");
    expect(totals.taxableGallons).toBe("0");
    expect(totals.lineTotal).toBe("0.00");
  });

  it("flags a line owed back to the fleet as a credit", () => {
    expect(isCreditLine(line({ lineTotal: "-4.50" }))).toBe(true);
    expect(isCreditLine(line({ lineTotal: "0.00" }))).toBe(false);
    expect(isCreditLine(line({ lineTotal: "4.50" }))).toBe(false);
  });
});

describe("groupLinesByFuelType", () => {
  it("groups by fuel type in first-seen order with a subtotal per group", () => {
    const groups = groupLinesByFuelType([
      line({ id: "a", fuelType: "Diesel", lineTotal: "1.00" }),
      line({ id: "b", fuelType: "Gasoline", lineTotal: "2.00" }),
      line({ id: "c", fuelType: "Diesel", lineTotal: "3.00" }),
    ]);
    expect(groups.map((group) => group.fuelType)).toEqual(["Diesel", "Gasoline"]);
    expect(groups[0].lines.map((entry) => entry.id)).toEqual(["a", "c"]);
    expect(groups[0].subtotal.lineTotal).toBe("4.00");
    expect(groups[1].subtotal.lineTotal).toBe("2.00");
  });
});

describe("linesMissingRates and canFinalize", () => {
  it("lists only member lines without a rate", () => {
    const missing = linesMissingRates([
      line({ id: "a", rateMissing: true }),
      line({ id: "b", rateMissing: false }),
      line({
        id: "c",
        rateMissing: true,
        isIftaMember: false,
        jurisdiction: {
          id: "ij_ak",
          countryCode: "US",
          code: "AK",
          name: "Alaska",
          isIftaMember: false,
          hasSurcharge: false,
        },
      }),
    ]);
    expect(missing.map((entry) => entry.id)).toEqual(["a"]);
  });

  it("allows finalizing only a Draft with every member rate present", () => {
    expect(canFinalize(ret())).toBe(true);
    expect(canFinalize(ret({ lines: [line({ rateMissing: true })] }))).toBe(false);
    expect(canFinalize(ret({ status: "Finalized" }))).toBe(false);
    expect(canFinalize(ret({ status: "Filed" }))).toBe(false);
  });
});

describe("returnTransitions", () => {
  it("offers Generate for a period without a return, only with Create", () => {
    expect(returnTransitions(null, ALL_PERMS)).toEqual(["generate"]);
    expect(returnTransitions(null, NO_PERMS)).toEqual([]);
  });

  it("offers recompute, finalize, delete and export on a Draft by permission", () => {
    expect(returnTransitions("Draft", ALL_PERMS)).toEqual([
      "recompute",
      "finalize",
      "delete",
      "export",
    ]);
    expect(returnTransitions("Draft", { ...NO_PERMS, approve: true })).toEqual(["finalize"]);
    expect(returnTransitions("Draft", { ...NO_PERMS, update: true, export: true })).toEqual([
      "recompute",
      "export",
    ]);
  });

  it("offers reopen, mark filed and export on a Finalized return", () => {
    expect(returnTransitions("Finalized", ALL_PERMS)).toEqual(["reopen", "markFiled", "export"]);
    expect(returnTransitions("Finalized", { ...NO_PERMS, submit: true })).toEqual(["markFiled"]);
    expect(returnTransitions("Finalized", { ...NO_PERMS, delete: true, update: true })).toEqual([]);
  });

  it("offers amend and export on a Filed return", () => {
    expect(returnTransitions("Filed", ALL_PERMS)).toEqual(["amend", "export"]);
    expect(returnTransitions("Filed", { ...NO_PERMS, create: true })).toEqual(["amend"]);
    expect(returnTransitions("Filed", { ...NO_PERMS, reopen: true, approve: true })).toEqual([]);
  });
});

describe("CSV export", () => {
  it("writes each line, a subtotal per fuel type and a grand total with signed values", () => {
    const rows = iftaReturnCsvRows(
      ret({
        lines: [
          line({ id: "a", fuelType: "Diesel", lineTotal: "2.00", taxDue: "2.00" }),
          line({
            id: "b",
            fuelType: "Diesel",
            lineTotal: "-5.25",
            taxDue: "-5.25",
            netTaxableGallons: "-25",
            jurisdiction: {
              id: "ij_ok",
              countryCode: "US",
              code: "OK",
              name: "Oklahoma",
              isIftaMember: true,
              hasSurcharge: false,
            },
          }),
          line({ id: "c", fuelType: "Gasoline", lineTotal: "1.00", taxDue: "1.00" }),
        ],
      }),
    );
    expect(rows.map((row) => row.kind)).toEqual([
      "line",
      "line",
      "subtotal",
      "line",
      "subtotal",
      "total",
    ]);
    expect(rows[1].jurisdiction).toBe("OK");
    expect(rows[1].lineTotal).toBe("-5.25");
    expect(rows[1].netTaxableGallons).toBe("-25");
    expect(rows[2].jurisdiction).toBe("Diesel subtotal");
    expect(rows[2].lineTotal).toBe("-3.25");
    expect(rows[4].lineTotal).toBe("1.00");
    expect(rows[5].jurisdiction).toBe("Total");
    expect(rows[5].lineTotal).toBe("-2.25");
    expect(rows[5].fuelType).toBe("");
  });

  it("exposes the columns the worksheet prints, in order", () => {
    expect(IFTA_RETURN_CSV_COLUMNS.map((column) => column.id)).toEqual([
      "fuelType",
      "jurisdiction",
      "member",
      "totalMiles",
      "taxableMiles",
      "taxPaidGallons",
      "taxableGallons",
      "netTaxableGallons",
      "ratePerGallon",
      "taxDue",
      "surchargeDue",
      "lineTotal",
    ]);
  });

  it("names the file after the period and the amendment", () => {
    expect(iftaReturnCsvFilename(ret())).toBe("ifta-return-2026q2.csv");
    expect(iftaReturnCsvFilename(ret({ amendmentNumber: 2 }))).toBe(
      "ifta-return-2026q2-amendment-2.csv",
    );
  });
});

describe("problemTone", () => {
  it("only a missing rate blocks the return; the rest are warnings or notes", () => {
    expect(problemTone("MissingRate")).toBe("danger");
    expect(problemTone("UnattributedMiles")).toBe("warn");
    expect(problemTone("NoTractorMiles")).toBe("warn");
    expect(problemTone("MileageMismatch")).toBe("warn");
    expect(problemTone("NoFuelForType")).toBe("warn");
    expect(problemTone("NonMemberActivity")).toBe("info");
    expect(problemTone("NonQualifiedActivity")).toBe("info");
  });
});

describe("formatIftaMoney", () => {
  it("prints a decimal string as currency at money scale", () => {
    expect(formatIftaMoney("1234.5", "USD")).toBe("$1,234.50");
    expect(formatIftaMoney("0", "USD")).toBe("$0.00");
  });

  it("keeps the sign so a credit reads as one", () => {
    expect(formatIftaMoney("-2.25", "USD")).toBe("-$2.25");
  });

  it("uses the return's own currency", () => {
    expect(formatIftaMoney("10.00", "CAD")).toBe("CA$10.00");
  });
});

describe("formatIftaMeasure", () => {
  it("rounds to the scale and groups thousands", () => {
    expect(formatIftaMeasure("12345.67", 0)).toBe("12,346");
    expect(formatIftaMeasure("6.254", 2)).toBe("6.25");
    expect(formatIftaMeasure("-1000", 0)).toBe("-1,000");
  });
});

describe("netPosition", () => {
  it("calls a positive net tax due and leaves it signed nowhere", () => {
    expect(netPosition("2.00")).toEqual({
      label: "Net tax due",
      magnitude: "2.00",
      isCredit: false,
    });
  });

  it("calls a negative net a credit and reports it as a magnitude", () => {
    expect(netPosition("-2.25")).toEqual({
      label: "Net credit",
      magnitude: "2.25",
      isCredit: true,
    });
  });

  it("treats zero as nothing due rather than a credit", () => {
    expect(netPosition("0.00")).toEqual({
      label: "Net tax due",
      magnitude: "0.00",
      isCredit: false,
    });
  });
});

describe("iftaYearOptions", () => {
  it("offers the current year and the four before it, newest first", () => {
    expect(iftaYearOptions(2026, 2026)).toEqual([2026, 2025, 2024, 2023, 2022]);
  });

  it("keeps a selected year that falls outside the window", () => {
    expect(iftaYearOptions(2026, 2019)).toEqual([2026, 2025, 2024, 2023, 2022, 2019]);
  });

  it("never offers a year the rate table cannot hold", () => {
    expect(iftaYearOptions(IFTA_MIN_YEAR, IFTA_MIN_YEAR)).toEqual([IFTA_MIN_YEAR]);
  });
});

describe("periodInclusiveEnd", () => {
  it("moves the exclusive bound back onto the quarter's last day", () => {
    // 2026-07-01T00:00:00Z is the exclusive end of Q2; the last day is Jun 30.
    expect(periodInclusiveEnd(1_782_864_000)).toBe(1_782_777_600);
  });
});

describe("problemTotals", () => {
  const problems = [
    { code: "MileageMismatch" as const, amount: "12.50" },
    { code: "MileageMismatch" as const, amount: "7.50" },
    { code: "MissingRate" as const, amount: null },
  ];

  it("counts the problems carrying a code and adds the figures they report", () => {
    expect(problemTotals(problems, "MileageMismatch")).toEqual({ count: 2, amount: "20.00" });
  });

  it("counts a problem that reports no figure without inventing one", () => {
    expect(problemTotals(problems, "MissingRate")).toEqual({ count: 1, amount: "0.00" });
  });

  it("is empty when the code never appears", () => {
    expect(problemTotals(problems, "NoTractorMiles")).toEqual({ count: 0, amount: "0.00" });
  });
});
