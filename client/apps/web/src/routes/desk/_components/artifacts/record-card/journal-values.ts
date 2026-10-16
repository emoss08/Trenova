import type { JournalCardLine, JournalCardPeriod } from "@/lib/graphql/journal-cards";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { CardStatusMark, CardTone } from "./record-card";
import { minorMoney } from "./record-card-values";

/** A journal's status as the card's pill draws it. */
export type Standing = { label: string; tone: CardTone; mark: CardStatusMark };

/** A journal's two sides added up, in minor units. */
export type JournalTotals = {
  debit: number;
  credit: number;
  /** Debits less credits: above zero the credits are short, below zero the debits are. */
  difference: number;
  balanced: boolean;
};

export function journalTotals(lines: readonly JournalCardLine[]): JournalTotals {
  let debit = 0;
  let credit = 0;
  for (const line of lines) {
    debit += line.debitAmount;
    credit += line.creditAmount;
  }
  return { debit, credit, difference: debit - credit, balanced: debit === credit };
}

/** An amount in minor units as a ledger column writes it: two places and no currency sign. */
export function ledgerAmount(minor: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(minor / 100);
}

/** An amount in minor units with the direction it moves a balance in: `+2,850.00`, `−312.00`. */
export function signedLedgerAmount(minor: number): string {
  if (minor === 0) return ledgerAmount(0);
  return `${minor > 0 ? "+" : "−"}${ledgerAmount(Math.abs(minor))}`;
}

/** The five terms of the accounting equation, in the order it is written. */
export const EQUATION_SIDES = ["assets", "liabilities", "equity", "revenue", "expenses"] as const;
export type EquationSide = (typeof EQUATION_SIDES)[number];

/** One account a journal moves, and by how much its balance grows (above zero) or shrinks. */
export type EquationAccount = { key: string; name: string; change: number };

/** What a journal does to one term of the equation: its balance's change and the accounts behind it. */
export type EquationTerm = { side: EquationSide; change: number; accounts: EquationAccount[] };

/** What a journal does to the books, term by term, with the lines whose account's side is not known. */
export type JournalEquation = {
  terms: Record<EquationSide, EquationTerm | null>;
  /** How many lines book to an account whose type could not be read. */
  unplaced: number;
};

const SIDE_OF: Readonly<Record<string, EquationSide>> = {
  Asset: "assets",
  Liability: "liabilities",
  Equity: "equity",
  Revenue: "revenue",
  CostOfRevenue: "expenses",
  Expense: "expenses",
};

/** Assets and expenses grow with a debit; liabilities, equity and revenue with a credit. */
const DEBIT_GROWS: ReadonlySet<EquationSide> = new Set(["assets", "expenses"]);

/**
 * What a journal does to each term of `Assets = Liabilities + Equity +
 * Revenue − Expenses`: each line moves its account's balance up or down by the
 * side its account sits on, and each term adds its accounts' moves together.
 * An account that ends where it started still lists, with no change.
 */
export function journalEquation(lines: readonly JournalCardLine[]): JournalEquation {
  const accounts = new Map<string, { side: EquationSide; account: EquationAccount }>();
  let unplaced = 0;
  for (const line of lines) {
    const side = line.category ? SIDE_OF[line.category] : undefined;
    if (!side) {
      unplaced += 1;
      continue;
    }
    const move = DEBIT_GROWS.has(side)
      ? line.debitAmount - line.creditAmount
      : line.creditAmount - line.debitAmount;
    const held = accounts.get(line.glAccountId);
    if (held) {
      held.account.change += move;
      continue;
    }
    accounts.set(line.glAccountId, {
      side,
      account: {
        key: line.glAccountId,
        name: line.accountName || line.accountCode || line.glAccountId,
        change: move,
      },
    });
  }

  const terms = {} as Record<EquationSide, EquationTerm | null>;
  for (const side of EQUATION_SIDES) terms[side] = null;
  for (const { side, account } of accounts.values()) {
    const term = terms[side] ?? { side, change: 0, accounts: [] };
    term.change += account.change;
    term.accounts.push(account);
    terms[side] = term;
  }
  for (const side of EQUATION_SIDES) {
    terms[side]?.accounts.sort((a, b) => Math.abs(b.change) - Math.abs(a.change));
  }
  return { terms, unplaced };
}

/** How much the left of the equation moves, and how much the right. */
export function equationSides(equation: JournalEquation): { left: number; right: number } {
  const change = (side: EquationSide) => equation.terms[side]?.change ?? 0;
  return {
    left: change("assets"),
    right: change("liabilities") + change("equity") + change("revenue") - change("expenses"),
  };
}

/** The sentence under the equation: whether the books still balance, and by how much each side moves. */
export function equationNote(
  totals: JournalTotals,
  equation: JournalEquation,
  t: TranslateFn,
  currency = "USD",
): { text: string; warn: boolean } {
  if (!totals.balanced) {
    return {
      text: t(
        "Debits and credits differ by {0}, so the books won't balance until the lines do.",
        minorMoney(Math.abs(totals.difference), currency),
      ),
      warn: true,
    };
  }
  if (equation.unplaced > 0) {
    return {
      text: t(
        "{0, plural, one {# line books to an account whose type could not be read, so it is left out.} other {# lines book to accounts whose types could not be read, so they are left out.}}",
        equation.unplaced,
      ),
      warn: false,
    };
  }
  const { left } = equationSides(equation);
  if (left === 0) {
    return {
      text: t("Assets don't move, and the other side nets to zero, so the books still balance."),
      warn: false,
    };
  }
  return {
    text:
      left > 0
        ? t(
            "Both sides of the equation rise by {0}, so the books still balance.",
            minorMoney(left, currency),
          )
        : t(
            "Both sides of the equation fall by {0}, so the books still balance.",
            minorMoney(-left, currency),
          ),
    warn: false,
  };
}

/** A posted or unposted journal line's other party, as the line's note gives it. */
export function lineNotes(line: JournalCardLine): string[] {
  const notes: string[] = [];
  if (line.description) notes.push(line.description);
  if (line.customerName) notes.push(line.customerName);
  if (line.locationName) notes.push(line.locationName);
  return notes;
}

/** A fiscal period's state as its chip says it. */
export function periodStateLabel(
  period: Pick<JournalCardPeriod, "status">,
  t: TranslateFn,
): string {
  switch (period.status) {
    case "Open":
      return t("Open for posting");
    case "Locked":
      return t("Locked");
    case "Closed":
      return t("Closed");
    case "PermanentlyClosed":
      return t("Closed for good");
    case "Inactive":
      return t("Not open yet");
    default:
      return period.status;
  }
}

/** A period that takes no more postings as things stand. */
export function periodIsShut(period: Pick<JournalCardPeriod, "status">): boolean {
  return period.status !== "Open" && period.status !== "Locked";
}

/**
 * An accounting date as a card writes it: the day it books on, read in UTC,
 * because the ledger keeps a booking day as that day's first second in UTC and
 * reading it in a reader's own timezone west of UTC would give the day before.
 */
export function accountingDay(seconds: number | null | undefined): string {
  if (!seconds) return "";
  return new Date(seconds * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  });
}

/** An accounting date as a calendar leaf draws it: the month over the day, read in UTC. */
export function accountingLeaf(seconds: number): { month: string; day: string } {
  const date = new Date(seconds * 1000);
  return {
    month: date.toLocaleDateString(undefined, { month: "short", timeZone: "UTC" }),
    day: date.toLocaleDateString(undefined, { day: "numeric", timeZone: "UTC" }),
  };
}
