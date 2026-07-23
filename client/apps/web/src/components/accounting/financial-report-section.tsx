import { AmountDisplay } from "@/components/accounting/amount-display";
import type { StatementSection } from "@/types/income-statement";
import { cn } from "@/lib/utils";

type FinancialReportSectionProps = {
  section: StatementSection | null | undefined;
  className?: string;
};

export function FinancialReportSection({ section, className }: FinancialReportSectionProps) {
  if (!section) return null;

  return (
    <div className={cn("space-y-1", className)}>
      <h3 className="text-sm font-semibold">{section.label}</h3>
      <div className="overflow-hidden rounded-md border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-left text-muted-foreground">
            <tr>
              <th className="px-3 py-2 text-xs font-medium">Account Code</th>
              <th className="px-3 py-2 text-xs font-medium">Account Name</th>
              <th className="px-3 py-2 text-right text-xs font-medium">Amount</th>
            </tr>
          </thead>
          <tbody>
            {section.lines.map((line) => (
              <tr
                key={line.accountCode}
                className="border-t transition-colors hover:bg-muted/50"
              >
                <td className="px-3 py-2 font-mono text-xs">{line.accountCode}</td>
                <td className="px-3 py-2 text-xs">{line.accountName}</td>
                <td className="px-3 py-2 text-right">
                  <AmountDisplay value={line.amountMinor} variant="auto" className="text-xs" />
                </td>
              </tr>
            ))}
          </tbody>
          <tfoot className="border-t bg-muted/30 font-medium">
            <tr>
              <td colSpan={2} className="px-3 py-2 text-right text-xs">
                Total {section.label}
              </td>
              <td className="px-3 py-2 text-right">
                <AmountDisplay
                  value={section.totalMinor}
                  variant="auto"
                  className="text-xs font-semibold"
                />
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}
