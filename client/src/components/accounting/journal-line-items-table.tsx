import { AmountDisplay } from "@/components/accounting/amount-display";
import type { JournalEntryLine } from "@/types/journal-entry";

type JournalLineItemsTableProps = {
  lines: JournalEntryLine[];
  totalDebit: number;
  totalCredit: number;
};

export function JournalLineItemsTable({ lines, totalDebit, totalCredit }: JournalLineItemsTableProps) {
  return (
    <div className="overflow-hidden rounded-md border">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-left text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-xs font-medium">Line</th>
            <th className="px-3 py-2 text-xs font-medium">Account</th>
            <th className="px-3 py-2 text-xs font-medium">Description</th>
            <th className="px-3 py-2 text-right text-xs font-medium">Debit</th>
            <th className="px-3 py-2 text-right text-xs font-medium">Credit</th>
          </tr>
        </thead>
        <tbody>
          {lines.map((line) => (
            <tr key={line.id} className="border-t transition-colors hover:bg-muted/50">
              <td className="px-3 py-2 font-mono text-2xs">{line.lineNumber}</td>
              <td className="px-3 py-2 font-mono text-xs">{line.glAccountId}</td>
              <td className="px-3 py-2 text-xs">{line.description}</td>
              <td className="px-3 py-2 text-right">
                {line.debitAmount > 0 ? (
                  <AmountDisplay value={line.debitAmount} className="text-xs" />
                ) : null}
              </td>
              <td className="px-3 py-2 text-right">
                {line.creditAmount > 0 ? (
                  <AmountDisplay value={line.creditAmount} className="text-xs" />
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot className="border-t bg-muted/30 font-medium">
          <tr>
            <td colSpan={3} className="px-3 py-2 text-right text-xs">
              Totals
            </td>
            <td className="px-3 py-2 text-right">
              <AmountDisplay value={totalDebit} className="text-xs font-semibold" />
            </td>
            <td className="px-3 py-2 text-right">
              <AmountDisplay value={totalCredit} className="text-xs font-semibold" />
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
  );
}
