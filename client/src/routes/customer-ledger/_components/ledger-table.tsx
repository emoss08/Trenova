import { AmountDisplay } from "@/components/accounting/amount-display";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { ARLedgerEntry } from "@/lib/graphql/accounts-receivable";
import { useMemo } from "react";

function formatDate(unix: number): string {
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function eventLabel(eventType: string): string {
  return eventType.replaceAll(/([a-z])([A-Z])/g, "$1 $2");
}

export function LedgerTable({ entries }: { entries: ARLedgerEntry[] }) {
  const rows = useMemo(() => {
    let runningBalance = 0;
    return entries.map((entry, index) => {
      runningBalance += entry.amountMinor;
      return { ...entry, runningBalance, key: `${entry.sourceObjectId}-${index}` };
    });
  }, [entries]);

  const totals = useMemo(
    () =>
      entries.reduce(
        (acc, entry) => ({
          charges: acc.charges + (entry.amountMinor > 0 ? entry.amountMinor : 0),
          payments: acc.payments + (entry.amountMinor < 0 ? -entry.amountMinor : 0),
        }),
        { charges: 0, payments: 0 },
      ),
    [entries],
  );

  const endingBalance = rows.length > 0 ? rows[rows.length - 1].runningBalance : 0;

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader className="bg-muted/50">
          <TableRow className="hover:bg-transparent">
            <TableHead className="h-9 text-xs">Date</TableHead>
            <TableHead className="h-9 text-xs">Document</TableHead>
            <TableHead className="h-9 text-xs">Event</TableHead>
            <TableHead className="h-9 text-xs">Source</TableHead>
            <TableHead className="h-9 text-right text-xs">Debit</TableHead>
            <TableHead className="h-9 text-right text-xs">Credit</TableHead>
            <TableHead className="h-9 text-right text-xs">Balance</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((entry) => (
            <TableRow key={entry.key} className="transition-colors">
              <TableCell className="py-2 text-xs">
                {formatDate(entry.transactionDate)}
              </TableCell>
              <TableCell className="py-2 font-mono text-xs">
                {entry.documentNumber || "—"}
              </TableCell>
              <TableCell className="py-2 text-xs text-muted-foreground">
                {eventLabel(entry.eventType)}
              </TableCell>
              <TableCell className="py-2">
                <SourceDrillDownLink
                  sourceType={entry.sourceObjectType}
                  sourceId={entry.sourceObjectId}
                />
              </TableCell>
              <TableCell className="py-2 text-right">
                {entry.amountMinor > 0 ? (
                  <AmountDisplay value={entry.amountMinor} className="text-xs" />
                ) : (
                  <span className="text-xs text-muted-foreground">—</span>
                )}
              </TableCell>
              <TableCell className="py-2 text-right">
                {entry.amountMinor < 0 ? (
                  <AmountDisplay
                    value={-entry.amountMinor}
                    className="text-xs text-green-600 dark:text-green-400"
                  />
                ) : (
                  <span className="text-xs text-muted-foreground">—</span>
                )}
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={entry.runningBalance} className="text-xs font-medium" />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
        <TableFooter className="bg-muted/40">
          <TableRow className="hover:bg-transparent">
            <TableCell colSpan={4} className="py-2 text-right text-xs font-medium">
              Totals · {rows.length} {rows.length === 1 ? "entry" : "entries"}
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={totals.charges} className="text-xs font-semibold" />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay
                value={totals.payments}
                className="text-xs font-semibold text-green-600 dark:text-green-400"
              />
            </TableCell>
            <TableCell className="py-2 text-right">
              <AmountDisplay value={endingBalance} className="text-xs font-bold" />
            </TableCell>
          </TableRow>
        </TableFooter>
      </Table>
    </div>
  );
}
