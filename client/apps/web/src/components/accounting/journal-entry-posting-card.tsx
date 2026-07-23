import { AccountingStatusBadge } from "@/components/accounting/accounting-status-badge";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { JournalLineItemsTable } from "@/components/accounting/journal-line-items-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";
import type { JournalEntryLine } from "@/types/journal-entry";
import { ChevronRightIcon, TriangleAlertIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";

export type PostingEntry = {
  id: string;
  entryNumber: string;
  entryType: string;
  status: string;
  accountingDate: number;
  description: string;
  totalDebit: number;
  totalCredit: number;
  isReversal: boolean;
  lines?: JournalEntryLine[] | null;
};

export function formatAccountingDate(unix: number | null | undefined): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function JournalEntryPostingCard({
  entry,
  defaultOpen = true,
}: {
  entry: PostingEntry;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const isBalanced = entry.totalDebit === entry.totalCredit;

  return (
    <div className="overflow-hidden rounded-md border bg-card">
      <div
        role="button"
        tabIndex={0}
        aria-expanded={open}
        onClick={() => setOpen((prev) => !prev)}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            setOpen((prev) => !prev);
          }
        }}
        className="flex cursor-pointer items-center gap-2.5 px-3 py-2.5 transition-colors select-none hover:bg-muted/40"
      >
        <ChevronRightIcon
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground transition-transform duration-200",
            open && "rotate-90",
          )}
        />
        <Link
          to={`/accounting/journal-entries/${entry.id}`}
          onClick={(event) => event.stopPropagation()}
          className="font-mono text-xs font-medium hover:underline"
        >
          {entry.entryNumber}
        </Link>
        <Badge variant="outline">{entry.entryType}</Badge>
        <AccountingStatusBadge status={entry.status} />
        {entry.isReversal ? <Badge variant="orange">Reversal</Badge> : null}
        {!isBalanced ? (
          <span className="inline-flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
            <TriangleAlertIcon className="size-3" />
            Out of balance
          </span>
        ) : null}
        <span className="ml-auto flex items-center gap-3 text-xs text-muted-foreground tabular-nums">
          <AmountDisplay value={entry.totalDebit} className="font-medium text-foreground" />
          <span>{formatAccountingDate(entry.accountingDate)}</span>
        </span>
      </div>
      {open ? (
        <div className="border-t px-3 pt-2.5 pb-3">
          {entry.description ? (
            <p className="mb-2 text-xs text-muted-foreground">{entry.description}</p>
          ) : null}
          {entry.lines?.length ? (
            <JournalLineItemsTable
              lines={entry.lines}
              totalDebit={entry.totalDebit}
              totalCredit={entry.totalCredit}
            />
          ) : (
            <p className="text-xs text-muted-foreground">No line detail available.</p>
          )}
        </div>
      ) : null}
    </div>
  );
}
