import type { JournalCardLine, JournalCardPeriod } from "@/lib/graphql/journal-cards";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment } from "react";
import {
  EQUATION_SIDES,
  ledgerAmount,
  lineNotes,
  periodIsShut,
  periodStateLabel,
  signedLedgerAmount,
  type EquationSide,
  type JournalEquation,
  type JournalTotals,
} from "./journal-values";
import { minorMoney } from "./record-card-values";
import { CardIcon, CardRecordRow } from "./record-card";

/**
 * A journal written into the book: each line by its account's code and name
 * with what it is for under it, debits first and the credits indented under
 * them, two ruled money columns and the totals over a double rule. A journal
 * that does not balance says by how much its short side falls short.
 */
export function CardJournal({
  lines,
  totals,
  label,
}: {
  lines: readonly JournalCardLine[];
  totals: JournalTotals;
  /** What the journal is, for a screen reader: an entry's lines, a request's. */
  label: string;
}) {
  const t = useT();
  const short = totals.difference > 0 ? "credit" : totals.difference < 0 ? "debit" : null;
  return (
    <table className="dk-rec-jrnl" aria-label={label}>
      <thead>
        <tr>
          <th className="dk-n" scope="col">
            <span className="sr-only">{t("Line")}</span>
            <span aria-hidden="true">#</span>
          </th>
          <th scope="col">{t("Account")}</th>
          <th className="dk-jm" scope="col">
            {t("Debit")}
          </th>
          <th className="dk-jm" scope="col">
            {t("Credit")}
          </th>
        </tr>
      </thead>
      <tbody>
        {lines.map((line) => {
          const notes = lineNotes(line);
          const credit = line.debitAmount === 0 && line.creditAmount > 0;
          return (
            <tr
              key={`${line.lineNumber}:${line.glAccountId}`}
              className={credit ? "dk-credit" : "dk-debit"}
            >
              <td className="dk-n">{line.lineNumber}</td>
              <td className="dk-acct">
                <span>
                  {line.accountCode && <code>{line.accountCode}</code>}
                  <b>{line.accountName || t("Account not found")}</b>
                </span>
                {notes.length > 0 && (
                  <em>
                    {notes.map((note) => (
                      <span key={note}>{note}</span>
                    ))}
                  </em>
                )}
              </td>
              <td className="dk-jm">
                {line.debitAmount > 0 ? ledgerAmount(line.debitAmount) : ""}
              </td>
              <td className="dk-jm">
                {line.creditAmount > 0 ? ledgerAmount(line.creditAmount) : ""}
              </td>
            </tr>
          );
        })}
      </tbody>
      <tfoot>
        <tr>
          <td />
          <th scope="row">{t("Totals")}</th>
          <td className={cn("dk-jm", short === "debit" && "dk-short")}>
            {ledgerAmount(totals.debit)}
          </td>
          <td className={cn("dk-jm", short === "credit" && "dk-short")}>
            {ledgerAmount(totals.credit)}
          </td>
        </tr>
        {short !== null && (
          <tr className="dk-off">
            <td />
            <th scope="row">
              {short === "credit" ? t("Credits are short by") : t("Debits are short by")}
            </th>
            <td className="dk-jm">{short === "debit" ? ledgerAmount(-totals.difference) : ""}</td>
            <td className="dk-jm">{short === "credit" ? ledgerAmount(totals.difference) : ""}</td>
          </tr>
        )}
      </tfoot>
    </table>
  );
}

/** Whether a journal balances, beside its title: an equals sign, or the amount it is off by. */
export function JournalSeal({ totals, currency }: { totals: JournalTotals; currency?: string }) {
  const t = useT();
  return totals.balanced ? (
    <span className="dk-rec-seal dk-ok">
      <i aria-hidden="true">=</i>
      {t("Debits equal credits")}
    </span>
  ) : (
    <span className="dk-rec-seal dk-warn">
      <i aria-hidden="true">≠</i>
      {t("Off by {0}", minorMoney(Math.abs(totals.difference), currency))}
    </span>
  );
}

const OPERATOR_AFTER: Readonly<Record<EquationSide, string>> = {
  assets: "=",
  liabilities: "+",
  equity: "+",
  revenue: "−",
  expenses: "",
};

/**
 * What a journal does to the books, as the accounting equation: each term in
 * its place with how much its balance rises or falls and the accounts behind
 * it, a term the journal leaves alone quiet, and a sentence under it saying
 * whether the books still balance.
 */
export function CardEquation({
  equation,
  note,
  currency,
}: {
  equation: JournalEquation;
  note: { text: string; warn: boolean };
  currency?: string;
}) {
  const t = useT();
  const names: Record<EquationSide, string> = {
    assets: t("Assets"),
    liabilities: t("Liabilities"),
    equity: t("Equity"),
    revenue: t("Revenue"),
    expenses: t("Expenses"),
  };
  return (
    <>
      <dl className="dk-rec-eqn" aria-label={t("What it does to the books")}>
        {EQUATION_SIDES.map((side) => {
          const term = equation.terms[side];
          const operator = OPERATOR_AFTER[side];
          return (
            <Fragment key={side}>
              {term ? (
                <div className="dk-moves">
                  <dt>{names[side]}</dt>
                  <dd>
                    {term.change === 0 ? (
                      t("Nets to zero")
                    ) : (
                      <>
                        <CardIcon name={term.change > 0 ? "rise" : "fall"} size={12} stroke={2.2} />
                        <span className="sr-only">{term.change > 0 ? t("Up") : t("Down")}</span>
                        {minorMoney(Math.abs(term.change), currency)}
                      </>
                    )}
                  </dd>
                  <dd className="dk-accts">
                    <ul>
                      {term.accounts.map((account) => (
                        <li key={account.key}>
                          <span>{account.name}</span>
                          <span>{signedLedgerAmount(account.change)}</span>
                        </li>
                      ))}
                    </ul>
                  </dd>
                </div>
              ) : (
                <div>
                  <dt>{names[side]}</dt>
                  <dd>{t("No change")}</dd>
                </div>
              )}
              {operator && <i aria-hidden="true">{operator}</i>}
            </Fragment>
          );
        })}
      </dl>
      <p className={cn("dk-rec-eqn-note", note.warn && "dk-warn")}>{note.text}</p>
    </>
  );
}

/**
 * A record a journal belongs to, as one row: its mark in a dashed stamp (INV,
 * PAY, JE), its number and a line saying what it is, opening in the card's
 * place when its kind has a card, else on its page, else not at all.
 */
export function JournalRefRow({
  mark,
  title,
  sub,
  to,
  onOpen,
}: {
  mark: string;
  title: string;
  sub: string;
  /** The record's page; null when it has none to open. */
  to: string | null;
  onOpen?: () => void;
}) {
  const body = (
    <>
      <span className="dk-rec-jref-mark" aria-hidden="true">
        {mark}
      </span>
      <span className="dk-rec-main">
        <b>{title}</b>
        <span className="dk-rec-sub">{sub}</span>
      </span>
      {(to || onOpen) && <CardIcon name="forward" size={12} />}
    </>
  );
  return to || onOpen ? (
    <CardRecordRow to={to ?? ""} onOpen={onOpen} className="dk-rec-jref">
      {body}
    </CardRecordRow>
  ) : (
    <div className="dk-rec-row dk-rec-jref">{body}</div>
  );
}

/** A fiscal period by its name with its state beside it, marked when it takes no more postings. */
export function PeriodChip({ period }: { period: Pick<JournalCardPeriod, "name" | "status"> }) {
  const t = useT();
  return (
    <span className="dk-rec-period">
      {period.name}
      <em className={cn(periodIsShut(period) && "dk-shut")}>{periodStateLabel(period, t)}</em>
    </span>
  );
}
