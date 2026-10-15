import { factoringAccountRoleLabel, factoringInvoiceEffectLabel } from "@/lib/factoring";
import type { FactoringPlannedMovement } from "@/lib/graphql/factoring";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

/**
 * The journal one factoring movement would book, drawn from the server's
 * plan: each line by the account it books to, the day it books on (moved to
 * the first open day when the asked day is in a closed period), whether it
 * waits for approval, and what it does to the invoice.
 */
export function FactoringMovementPreview({
  movement,
  currencyCode,
  showInvoice = false,
}: {
  movement: FactoringPlannedMovement;
  currencyCode: string;
  showInvoice?: boolean;
}) {
  const t = useT();
  const money = (minor: number) => formatMinor(minor, currencyCode);

  if (movement.lines.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {showInvoice
          ? t("Nothing is booked for invoice {0}.", movement.invoiceNumber)
          : t("Nothing is booked: no money moves.")}
      </p>
    );
  }

  const totalDebit = movement.lines.reduce((sum, line) => sum + line.debit, 0);
  const totalCredit = movement.lines.reduce((sum, line) => sum + line.credit, 0);
  const redated = movement.accountingDate !== movement.requestedDate;

  return (
    <div className="flex flex-col gap-2">
      {showInvoice ? (
        <h4 className="text-sm font-medium">{t("Invoice {0}", movement.invoiceNumber)}</h4>
      ) : null}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("Account")}</TableHead>
            <TableHead className="text-right">{t("Debit")}</TableHead>
            <TableHead className="text-right">{t("Credit")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {movement.lines.map((line) => (
            <TableRow key={`${line.role}-${line.accountId}-${line.debit}-${line.credit}`}>
              <TableCell>
                <div className="flex flex-col">
                  <span>{factoringAccountRoleLabel(line.role)}</span>
                  <span className="text-muted-foreground text-xs">
                    {line.account
                      ? `${line.account.accountCode} · ${line.account.name}`
                      : line.accountId}
                  </span>
                </div>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {line.debit > 0 ? money(line.debit) : ""}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {line.credit > 0 ? money(line.credit) : ""}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
        <TableFooter>
          <TableRow>
            <TableCell>{t("Total")}</TableCell>
            <TableCell className="text-right tabular-nums">{money(totalDebit)}</TableCell>
            <TableCell className="text-right tabular-nums">{money(totalCredit)}</TableCell>
          </TableRow>
        </TableFooter>
      </Table>
      <ul className="text-muted-foreground flex flex-col gap-0.5 text-xs">
        <li>
          {redated
            ? t(
                "{0} is in a closed period, so it books on {1}, the first open day.",
                formatUnixDateMedium(movement.requestedDate),
                formatUnixDateMedium(movement.accountingDate),
              )
            : t("Books on {0}.", formatUnixDateMedium(movement.accountingDate))}
          {movement.fiscalPeriodName ? ` ${t("Period {0}.", movement.fiscalPeriodName)}` : ""}
        </li>
        {movement.requiresApproval ? (
          <li>{t("The entry waits for approval before it reaches the ledger.")}</li>
        ) : null}
        <li>{factoringInvoiceEffectLabel(movement.movement.invoiceEffect)}</li>
      </ul>
    </div>
  );
}
