import { describe, expect, it } from "vitest";
import {
  disputeCases,
  fallScale,
  headFacts,
  lineBasis,
  settlementFigure,
  settlementFlags,
  settlementKicker,
  settlementSubtitle,
  statementGroups,
  waterfall,
  workerName,
} from "../driver-settlement-card";
import {
  DAY,
  dispute,
  emptySettlement,
  group,
  HOUR,
  paidSettlement,
  pendingSettlement,
  shortSettlement,
  t,
  voidedSettlement,
} from "./fixtures";

describe("the head", () => {
  it("names the worker, the period and how they are paid", () => {
    const record = pendingSettlement();
    expect(workerName(record, t)).toBe("Rosa Martinez");
    expect(settlementKicker(record, t)).toEqual(["DS-2610-0142", "Oct 5 – Oct 11", "Company"]);
    expect(settlementSubtitle(record, t)).toBe(
      "Pay for 9 loads · 2,140 mi · Company OTR – mileage bands",
    );
  });

  it("falls back to the worker's names, then to no name, when the whole name is blank", () => {
    expect(
      workerName(
        pendingSettlement({
          worker: { id: "wrk_1", firstName: "Ana", lastName: "Silva", wholeName: " " },
        }),
        t,
      ),
    ).toBe("Ana Silva");
    expect(workerName(pendingSettlement({ worker: null }), t)).toBe("Unknown worker");
  });

  it("says a period with no loads and no miles", () => {
    expect(settlementSubtitle(emptySettlement(), t)).toBe(
      "No loads this period · Company OTR – mileage bands",
    );
  });

  it("reads the net pay as what a pending settlement pays and when", () => {
    expect(settlementFigure(pendingSettlement(), t)).toEqual({
      amount: "$1,684.20",
      caption: expect.stringMatching(/^net pay · pays \w+, Oct 17$/u),
      tone: "plain",
    });
  });

  it("says what carries forward when deductions ran past earnings", () => {
    expect(settlementFigure(shortSettlement(), t)).toEqual({
      amount: "$0.00",
      caption: "net pay · $212.40 carries forward",
      tone: "zero",
    });
  });

  it("reads a paid settlement by its payment, and strikes a voided one", () => {
    expect(settlementFigure(paidSettlement(), t).caption).toBe("net pay · paid Oct 10");
    expect(settlementFigure(voidedSettlement(), t)).toMatchObject({
      caption: "net pay · voided",
      tone: "struck",
    });
  });

  it("counts the flags and open disputes beside the pill, and how a paid one went out", () => {
    expect(headFacts(pendingSettlement(), t)).toEqual(["2 flags to review", "1 open dispute"]);
    expect(headFacts(paidSettlement(), t)).toEqual(["ACH · 0219 4471 2208"]);
  });
});

describe("settlementFlags", () => {
  it("titles each flag by its code, keeps the server's words and puts critical ones first", () => {
    const flags = settlementFlags(
      pendingSettlement({
        exceptions: [
          { code: "ManualAdjustment", severity: "Warning", message: "Added by hand." },
          { code: "NegativeNet", severity: "Critical", message: "Short $212.40." },
          { code: "SomethingNew", severity: "Warning", message: "A new check." },
        ],
      }),
      t,
    );
    expect(flags).toEqual([
      {
        key: "NegativeNet",
        severity: "critical",
        title: "Deductions ran past earnings",
        body: "Short $212.40.",
      },
      {
        key: "ManualAdjustment",
        severity: "warning",
        title: "Pay added by hand",
        body: "Added by hand.",
      },
      {
        key: "SomethingNew",
        severity: "warning",
        title: "Review before approving",
        body: "A new check.",
      },
    ]);
  });

  it("reads no flags when the server sent none", () => {
    expect(settlementFlags(pendingSettlement({ exceptions: null }), t)).toEqual([]);
  });
});

describe("waterfall", () => {
  it("runs from earnings through reimbursements and deductions to the net pay", () => {
    const steps = waterfall(pendingSettlement(), t);
    expect(steps.map((step) => [step.key, step.from, step.to, step.sign])).toEqual([
      ["earnings", 0, 188_680, "in"],
      ["reimbursements", 188_680, 201_420, "in"],
      ["deductions", 168_420, 201_420, "out"],
      ["net", 0, 168_420, "total"],
    ]);
    expect(steps[0]?.note).toBe("9 loads");
  });

  it("hangs a shortfall below zero and widens the scale to hold it", () => {
    const steps = waterfall(shortSettlement(), t);
    expect(steps.map((step) => [step.key, step.from, step.to])).toEqual([
      ["earnings", 0, 258_720],
      ["deductions", -21_240, 258_720],
      ["net", 0, 0],
      ["carriesForward", -21_240, 0],
    ]);
    expect(fallScale(steps)).toEqual({ low: -21_240, high: 258_720 });
  });

  it("draws a carried-in shortfall as taken off", () => {
    const steps = waterfall(
      pendingSettlement({ carryForwardInMinor: -5_000, netPayMinor: 163_420 }),
      t,
    );
    expect(steps.find((step) => step.key === "carriedIn")).toMatchObject({
      sign: "out",
      from: 196_420,
      to: 201_420,
    });
  });

  it("keeps a scale of nothing from dividing by zero", () => {
    expect(fallScale(waterfall(emptySettlement(), t))).toEqual({ low: 0, high: 1 });
  });
});

describe("statementGroups", () => {
  it("groups the rows under what each does to the net pay, with each group's total", () => {
    const groups = statementGroups(pendingSettlement(), t);
    expect(groups.map((item) => [item.title, item.total, item.rows.length])).toEqual([
      ["Earnings", "$1,886.80", 4],
      ["Reimbursements", "$127.40", 2],
      ["Deductions", "−$330.00", 3],
    ]);
    expect(
      groups[0]?.rows.map((row) => [row.name, row.detail, row.basis, row.amount, row.manual]),
    ).toEqual([
      ["Linehaul", "9 loads", "2,140 mi × $0.62", "$1,326.80", false],
      ["Stop pay", "9 loads", "14 stops × $25.00", "$350.00", false],
      ["Detention", "2 loads", "3 hr × $20.00", "$60.00", false],
      ["Layover - Detroit 10/8", "", "Flat", "$150.00", true],
    ]);
    expect(groups[2]?.rows.map((row) => [row.name, row.detail, row.basis])).toEqual([
      ["Advance ADV-0088", "$300.00 still owed on the advance", "$200.00 a settlement"],
      ["Health insurance", "Every settlement", "Flat"],
      ["Phone plan", "Every settlement", "Flat"],
    ]);
  });

  it("files a negative adjustment under deductions and names a nameless line by its kind", () => {
    const groups = statementGroups(
      pendingSettlement({
        lineSummary: [
          group({ category: "Adjustment", amountMinor: -2_500, manual: true }),
          group({ category: "EscrowContribution", amountMinor: -1_000 }),
        ],
      }),
      t,
    );
    expect(groups.map((item) => item.key)).toEqual(["deductions"]);
    expect(groups[0]?.rows.map((row) => row.name)).toEqual(["Adjustment", "Escrow"]);
  });

  it("reads an escrow row by what it holds against its target", () => {
    const groups = statementGroups(shortSettlement(), t);
    const escrow = groups.at(-1)?.rows.at(-1);
    expect(escrow?.detail).toBe("$1,900.00 of $2,500.00 held");
  });

  it("reads no groups for a settlement with no lines", () => {
    expect(statementGroups(emptySettlement(), t)).toEqual([]);
    expect(statementGroups(pendingSettlement({ lineSummary: null }), t)).toEqual([]);
  });
});

describe("lineBasis", () => {
  it("works each method out in its own units, and says the units alone when rates differ", () => {
    expect(lineBasis(group({ method: "PercentOfRevenue", rate: "72" }), "USD", t)).toBe(
      "72% of revenue",
    );
    expect(lineBasis(group({ method: "FlatPerShipment", rate: "50" }), "USD", t)).toBe(
      "$50.00 a load",
    );
    expect(lineBasis(group({ method: "PerTotalMile", quantity: "980.5" }), "USD", t)).toBe(
      "980.5 mi",
    );
    expect(lineBasis(group({ method: "PerStop", quantity: "1", rate: "25" }), "USD", t)).toBe(
      "1 stop × $25.00",
    );
    expect(lineBasis(group({ method: "PerDay", quantity: "2" }), "USD", t)).toBe("2 days");
    expect(lineBasis(group({ category: "Reimbursement" }), "USD", t)).toBe("As receipted");
  });
});

describe("disputeCases", () => {
  it("puts the open disputes first, each with how long it has been open", () => {
    const cases = disputeCases(
      pendingSettlement({
        disputes: [
          dispute({
            id: "dsd_old",
            status: "Denied",
            resolutionNote: "The stop was not on the bill of lading.",
            resolvedAt: DAY.oct10,
            createdAt: DAY.oct6,
          }),
          dispute(),
        ],
      }),
      DAY.oct14 + 15 * HOUR,
      t,
    );
    expect(cases.map((item) => [item.id, item.open, item.status])).toEqual([
      ["dsd_1", true, "In review"],
      ["dsd_old", false, "Denied"],
    ]);
    expect(cases[0]?.title).toBe("Missing pay · Stop pay · S-240201");
    expect(cases[0]?.meta).toMatch(/^Raised Oct 12, .+ · 1 day open$/u);
    expect(cases[1]?.body).toBe("The stop was not on the bill of lading.");
    expect(cases[1]?.meta).toBe("Closed Oct 10");
  });

  it("names an advance only where the line's own name does not", () => {
    const advance = {
      id: "padv_9",
      reference: "ADV-0091",
      amountMinor: 40_000,
      outstandingMinor: 25_000,
    };
    const rows = (description: string) =>
      statementGroups(
        pendingSettlement({
          lineSummary: [
            group({
              category: "AdvanceRecovery",
              description,
              amountMinor: -15_000,
              advanceId: advance.id,
              advance,
            }),
          ],
        }),
        t,
      )[0]?.rows.map((row) => [row.name, row.detail]);
    expect(rows("Fuel advance")).toEqual([
      ["Fuel advance", "Advance ADV-0091 · $250.00 still owed"],
    ]);
    expect(rows("Advance ADV-0091")).toEqual([
      ["Advance ADV-0091", "$250.00 still owed on the advance"],
    ]);
  });

  it("says when a dispute raised less than a day ago was raised, without counting days", () => {
    const [raised] = disputeCases(
      pendingSettlement({ disputes: [dispute({ createdAt: DAY.oct14 + 9 * HOUR })] }),
      DAY.oct14 + 15 * HOUR,
      t,
    );
    expect(raised?.meta).toMatch(/^Raised Oct 14, [^·]+$/u);
  });
});
