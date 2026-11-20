import { useT } from "@trenova/shared/i18n/use-t";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import type { ARLedgerEntry } from "@/lib/graphql/accounts-receivable";
import { useEffect, useRef } from "react";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

function formatDate(unix: number): string {
  return formatUnixDateMedium(unix);
}

function eventLabel(eventType: string): string {
  return eventType.replaceAll(/([a-z])([A-Z])/g, "$1 $2");
}

export function LedgerTable({
  entries,
  hasNextPage,
  isFetchingNextPage,
  onLoadMore,
}: {
  entries: ARLedgerEntry[];
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  onLoadMore: () => void;
}) {
  const t = useT();
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const target = endRef.current;
    if (!target || !hasNextPage) return;
    const observer = new IntersectionObserver(
      (observed) => {
        if (observed[0]?.isIntersecting && !isFetchingNextPage) {
          onLoadMore();
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, onLoadMore]);

  const totals = entries.reduce(
    (acc, entry) => ({
      charges: acc.charges + (entry.amountMinor > 0 ? entry.amountMinor : 0),
      payments: acc.payments + (entry.amountMinor < 0 ? -entry.amountMinor : 0),
    }),
    { charges: 0, payments: 0 },
  );
  const endingBalance = entries.length > 0 ? entries[entries.length - 1].runningBalanceMinor : 0;

  return (
    <ScrollArea className="max-h-160 rounded-md border" maskHeight={18}>
      <Table>
        <TableHeader className="bg-muted/50">
          <TableRow className="hover:bg-transparent">
            <TableHead className="h-9 text-xs">{t("Date")}</TableHead>
            <TableHead className="h-9 text-xs">{t("Document")}</TableHead>
            <TableHead className="h-9 text-xs">{t("Event")}</TableHead>
            <TableHead className="h-9 text-xs">{t("Source")}</TableHead>
            <TableHead className="h-9 text-right text-xs">{t("Debit")}</TableHead>
            <TableHead className="h-9 text-right text-xs">{t("Credit")}</TableHead>
            <TableHead className="h-9 text-right text-xs">{t("Balance")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry, index) => (
            <TableRow key={`${entry.sourceObjectId}-${index}`} className="transition-colors">
              <TableCell className="py-2 text-xs">{formatDate(entry.transactionDate)}</TableCell>
              <TableCell className="py-2 font-mono text-xs">
                {entry.documentNumber || "—"}
              </TableCell>
              <TableCell className="text-muted-foreground py-2 text-xs">
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
                  <span className="text-muted-foreground text-xs">—</span>
                )}
              </TableCell>
              <TableCell className="py-2 text-right">
                {entry.amountMinor < 0 ? (
                  <AmountDisplay
                    value={-entry.amountMinor}
                    className="text-xs text-success-foreground"
                  />
                ) : (
                  <span className="text-muted-foreground text-xs">—</span>
                )}
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={entry.runningBalanceMinor} className="text-xs font-medium" />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
        {hasNextPage ? null : (
          <TableFooter className="bg-muted/40">
            <TableRow className="hover:bg-transparent">
              <TableCell colSpan={4} className="py-2 text-right text-xs font-medium">
                {t("Totals · {0, plural, one {# entry} other {# entries}}", entries.length)}
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={totals.charges} className="text-xs font-semibold" />
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay
                  value={totals.payments}
                  className="text-xs font-semibold text-success-foreground"
                />
              </TableCell>
              <TableCell className="py-2 text-right">
                <AmountDisplay value={endingBalance} className="text-xs font-semibold" />
              </TableCell>
            </TableRow>
          </TableFooter>
        )}
      </Table>
      {isFetchingNextPage ? (
        <div className="flex items-center justify-center py-3">
          <TextShimmer className="text-xs" duration={1}>
            {t("Loading more...")}
          </TextShimmer>
        </div>
      ) : null}
      {hasNextPage ? <div ref={endRef} className="h-px" /> : null}
    </ScrollArea>
  );
}
