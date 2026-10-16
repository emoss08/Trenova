import type { JournalCardLine, JournalCardPeriod } from "@/lib/graphql/journal-cards";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { afterEach, describe, expect, it } from "vitest";
import {
  accountingDay,
  accountingLeaf,
  equationNote,
  equationSides,
  journalEquation,
  journalTotals,
  ledgerAmount,
  lineNotes,
  periodIsShut,
  periodStateLabel,
  signedLedgerAmount,
} from "../journal-values";

const t = translate as TranslateFn;

function line(
  lineNumber: number,
  account: string,
  category: string | null,
  debit: number,
  credit: number,
  extra: Partial<JournalCardLine> = {},
): JournalCardLine {
  return {
    lineNumber,
    glAccountId: `gla_${account}`,
    accountCode: account,
    accountName: `Account ${account}`,
    category,
    description: "",
    debitAmount: debit,
    creditAmount: credit,
    customerId: null,
    customerName: "",
    locationId: null,
    locationName: "",
    ...extra,
  } as JournalCardLine;
}

const INVOICE = [
  line(1, "1200", "Asset", 361250, 0),
  line(2, "4000", "Revenue", 0, 285000),
  line(3, "4010", "Revenue", 0, 51250),
  line(4, "4100", "Revenue", 0, 25000),
];

const SETTLEMENT = [
  line(1, "5100", "CostOfRevenue", 258720, 0),
  line(2, "1330", "Asset", 21240, 0),
  line(3, "1320", "Asset", 0, 269960),
  line(4, "2110", "Liability", 0, 10000),
];

describe("journalTotals", () => {
  it("adds each side and reads a journal whose sides agree as balanced", () => {
    expect(journalTotals(INVOICE)).toEqual({
      debit: 361250,
      credit: 361250,
      difference: 0,
      balanced: true,
    });
  });

  it("gives debits less credits, positive when the credits fall short", () => {
    expect(
      journalTotals([line(1, "5400", "Expense", 126000, 0), line(2, "5350", "Expense", 0, 122000)]),
    ).toEqual({ debit: 126000, credit: 122000, difference: 4000, balanced: false });
    expect(
      journalTotals([line(1, "5400", "Expense", 100, 0), line(2, "1200", "Asset", 0, 300)]),
    ).toMatchObject({ difference: -200, balanced: false });
  });

  it("reads a journal with no lines as balanced at zero", () => {
    expect(journalTotals([])).toEqual({ debit: 0, credit: 0, difference: 0, balanced: true });
  });
});

describe("ledger amounts", () => {
  it("writes minor units with two places and no sign of currency", () => {
    expect(ledgerAmount(285000)).toBe("2,850.00");
    expect(ledgerAmount(0)).toBe("0.00");
  });

  it("marks which way a balance moves with a true minus sign", () => {
    expect(signedLedgerAmount(285000)).toBe("+2,850.00");
    expect(signedLedgerAmount(-31200)).toBe("−312.00");
    expect(signedLedgerAmount(0)).toBe("0.00");
  });
});

describe("journalEquation", () => {
  it("raises assets with a debit and revenue with a credit", () => {
    const equation = journalEquation(INVOICE);
    expect(equation.unplaced).toBe(0);
    expect(equation.terms.assets).toEqual({
      side: "assets",
      change: 361250,
      accounts: [{ key: "gla_1200", name: "Account 1200", change: 361250 }],
    });
    expect(equation.terms.revenue?.change).toBe(361250);
    expect(equation.terms.revenue?.accounts.map((account) => account.key)).toEqual([
      "gla_4000",
      "gla_4010",
      "gla_4100",
    ]);
    expect(equation.terms.liabilities).toBeNull();
    expect(equation.terms.equity).toBeNull();
    expect(equation.terms.expenses).toBeNull();
    expect(equationSides(equation)).toEqual({ left: 361250, right: 361250 });
  });

  it("counts cost of revenue as an expense and nets an asset debited and credited", () => {
    const equation = journalEquation(SETTLEMENT);
    expect(equation.terms.expenses).toEqual({
      side: "expenses",
      change: 258720,
      accounts: [{ key: "gla_5100", name: "Account 5100", change: 258720 }],
    });
    expect(equation.terms.assets?.change).toBe(21240 - 269960);
    expect(equation.terms.assets?.accounts).toEqual([
      { key: "gla_1320", name: "Account 1320", change: -269960 },
      { key: "gla_1330", name: "Account 1330", change: 21240 },
    ]);
    expect(equation.terms.liabilities?.change).toBe(10000);
    expect(equationSides(equation)).toEqual({ left: -248720, right: -248720 });
  });

  it("adds two lines on one account together and keeps an account that ends where it began", () => {
    const equation = journalEquation([
      line(1, "5300", "Expense", 1842000, 0),
      line(2, "5300", "Expense", 611550, 0),
      line(3, "2200", "Liability", 0, 2453550),
      line(4, "1000", "Asset", 5000, 0),
      line(5, "1000", "Asset", 0, 5000),
    ]);
    expect(equation.terms.expenses?.accounts).toEqual([
      { key: "gla_5300", name: "Account 5300", change: 2453550 },
    ]);
    expect(equation.terms.assets).toEqual({
      side: "assets",
      change: 0,
      accounts: [{ key: "gla_1000", name: "Account 1000", change: 0 }],
    });
  });

  it("leaves out a line whose account type could not be read, and counts it", () => {
    const equation = journalEquation([
      line(1, "1200", "Asset", 500, 0),
      line(2, "9999", null, 0, 500),
      line(3, "9998", "Unknown", 0, 0),
    ]);
    expect(equation.unplaced).toBe(2);
    expect(equation.terms.assets?.change).toBe(500);
  });

  it("names an account by its code, then its id, when its name is missing", () => {
    const equation = journalEquation([
      line(1, "1200", "Asset", 500, 0, { accountName: "" }),
      line(2, "", "Revenue", 0, 500, { accountName: "", accountCode: "" }),
    ]);
    expect(equation.terms.assets?.accounts[0]?.name).toBe("1200");
    expect(equation.terms.revenue?.accounts[0]?.name).toBe("gla_");
  });
});

describe("equationNote", () => {
  it("says both sides rise by the amount when the journal grows the books", () => {
    expect(equationNote(journalTotals(INVOICE), journalEquation(INVOICE), t)).toEqual({
      text: "Both sides of the equation rise by $3,612.50, so the books still balance.",
      warn: false,
    });
  });

  it("says both sides fall when the journal shrinks them, in the journal's currency", () => {
    const lines = [line(1, "6900", "Expense", 25000, 0), line(2, "1200", "Asset", 0, 25000)];
    expect(equationNote(journalTotals(lines), journalEquation(lines), t, "CAD")).toEqual({
      text: "Both sides of the equation fall by CA$250.00, so the books still balance.",
      warn: false,
    });
  });

  it("says nothing moves when assets are untouched", () => {
    const lines = [line(1, "5400", "Expense", 1200, 0), line(2, "5350", "Expense", 0, 1200)];
    expect(equationNote(journalTotals(lines), journalEquation(lines), t)).toEqual({
      text: "Assets don't move, and the other side nets to zero, so the books still balance.",
      warn: false,
    });
  });

  it("warns by how much the sides differ before anything else", () => {
    const lines = [line(1, "5400", "Expense", 126000, 0), line(2, "5350", null, 0, 122000)];
    expect(equationNote(journalTotals(lines), journalEquation(lines), t)).toEqual({
      text: "Debits and credits differ by $40.00, so the books won't balance until the lines do.",
      warn: true,
    });
  });

  it("counts the lines it had to leave out, one or many", () => {
    const one = [line(1, "1200", "Asset", 500, 0), line(2, "9999", null, 0, 500)];
    expect(equationNote(journalTotals(one), journalEquation(one), t).text).toBe(
      "1 line books to an account whose type could not be read, so it is left out.",
    );
    const two = [line(1, "1200", null, 500, 0), line(2, "9999", null, 0, 500)];
    expect(equationNote(journalTotals(two), journalEquation(two), t).text).toBe(
      "2 lines book to accounts whose types could not be read, so they are left out.",
    );
  });
});

describe("lineNotes", () => {
  it("gives the line's description, customer and location in that order, skipping blanks", () => {
    expect(
      lineNotes(
        line(1, "1200", "Asset", 1, 0, {
          description: "Invoice INV-2610-0042",
          customerName: "Acme Foods",
          locationName: "Plant 2",
        }),
      ),
    ).toEqual(["Invoice INV-2610-0042", "Acme Foods", "Plant 2"]);
    expect(lineNotes(line(1, "1200", "Asset", 1, 0, { customerName: "Acme Foods" }))).toEqual([
      "Acme Foods",
    ]);
  });
});

describe("periods", () => {
  it("names each period state and reads only open and locked periods as taking postings", () => {
    const states: JournalCardPeriod["status"][] = [
      "Open",
      "Locked",
      "Closed",
      "PermanentlyClosed",
      "Inactive",
    ];
    expect(states.map((status) => periodStateLabel({ status }, t))).toEqual([
      "Open for posting",
      "Locked",
      "Closed",
      "Closed for good",
      "Not open yet",
    ]);
    expect(states.map((status) => periodIsShut({ status }))).toEqual([
      false,
      false,
      true,
      true,
      true,
    ]);
  });
});

describe("accounting dates", () => {
  const zone = process.env.TZ;

  afterEach(() => {
    process.env.TZ = zone;
  });

  it("reads a booking day in UTC, so a reader west of UTC sees the day it books on", () => {
    process.env.TZ = "America/Los_Angeles";
    const firstOfOctober = Date.UTC(2026, 9, 1) / 1000;

    expect(new Date(firstOfOctober * 1000).getDate()).toBe(30);
    expect(accountingDay(firstOfOctober)).toBe("Oct 1, 2026");
    expect(accountingLeaf(firstOfOctober)).toEqual({ month: "Oct", day: "1" });
  });

  it("writes nothing for a journal with no accounting date", () => {
    expect(accountingDay(null)).toBe("");
    expect(accountingDay(0)).toBe("");
  });
});
