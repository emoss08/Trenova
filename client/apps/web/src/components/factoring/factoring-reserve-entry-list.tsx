import { recordPath } from "@/config/record-links";
import { FACTORING_RESERVE_ENTRY_KIND_LABELS } from "@/lib/factoring";
import type { FactoringReserveEntry } from "@/lib/graphql/factoring";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Link } from "react-router";

/**
 * Reserve movements as rows: when each happened, what it was, the invoice or
 * schedule it belongs to, the cash that moved (in from the factor or paid
 * back to it), the fee, and the journal entry it booked.
 */
export function FactoringReserveEntryList({
  entries,
  show,
}: {
  entries: readonly FactoringReserveEntry[];
  show: "invoice" | "schedule";
}) {
  const t = useT();

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("Date")}</TableHead>
          <TableHead>{t("Movement")}</TableHead>
          <TableHead>{show === "invoice" ? t("Invoice #") : t("Schedule")}</TableHead>
          <TableHead className="text-right">{t("Cash")}</TableHead>
          <TableHead className="text-right">{t("Fee")}</TableHead>
          <TableHead>{t("Journal")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {entries.map((entry) => (
          <TableRow key={entry.id}>
            <TableCell>{formatUnixDateMedium(entry.occurredOn)}</TableCell>
            <TableCell>
              <div className="flex flex-col">
                <span>{FACTORING_RESERVE_ENTRY_KIND_LABELS[entry.kind]}</span>
                {entry.memo ? (
                  <span className="text-muted-foreground text-xs">{entry.memo}</span>
                ) : null}
              </div>
            </TableCell>
            <TableCell>
              {show === "invoice" ? (
                <Link
                  to={recordPath("invoice", entry.invoiceId)}
                  className="font-mono hover:underline"
                >
                  {entry.invoice?.number ?? entry.invoiceId}
                </Link>
              ) : (
                <Link
                  to={recordPath("factoring_submission", entry.submissionId)}
                  className="font-mono hover:underline"
                >
                  {entry.submission?.number ?? entry.submissionId}
                </Link>
              )}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatMinor(entry.cashMinor, entry.currencyCode)}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {formatMinor(entry.feeMinor, entry.currencyCode)}
            </TableCell>
            <TableCell>
              <Link
                to={recordPath("journal_entry", entry.journalEntryId)}
                className="text-brand hover:underline"
              >
                {t("View")}
              </Link>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
