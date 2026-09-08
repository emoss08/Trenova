import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useState } from "react";

const TRIAL_BALANCE_COLUMNS = [
  { label: "Account" },
  { label: "Name" },
  { label: "Category" },
  { label: "Debit", numeric: true },
  { label: "Credit", numeric: true },
  { label: "Net change", numeric: true },
] as const;

export function TrialBalancePage() {
  const [periodId, setPeriodId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    ...queries.accountingReport.trialBalance(periodId!),
    enabled: Boolean(periodId),
  });

  const balances = data ?? [];

  const totalDebit = balances.reduce((sum, b) => sum + b.periodDebitMinor, 0);
  const totalCredit = balances.reduce((sum, b) => sum + b.periodCreditMinor, 0);
  const totalNet = balances.reduce((sum, b) => sum + b.netChangeMinor, 0);

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Trial Balance",
        description: "View account balances for a fiscal period.",
      }}
      className="p-0"
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <EmptyTable
            title="Pick a period"
            description="Choose a fiscal period above and every account's debits, credits and net change for it are listed here."
            columns={TRIAL_BALANCE_COLUMNS}
          />
        ) : isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : balances.length === 0 ? (
          <EmptyTable
            title="Nothing posted"
            description="No journal entry landed in this period, so every account stands where it did. Post one, or pick another period."
            columns={TRIAL_BALANCE_COLUMNS}
          />
        ) : (
          <div className="overflow-hidden rounded-md border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-muted-foreground text-left">
                <tr>
                  <th className="px-3 py-2 text-xs font-medium">Account Code</th>
                  <th className="px-3 py-2 text-xs font-medium">Account Name</th>
                  <th className="px-3 py-2 text-xs font-medium">Category</th>
                  <th className="px-3 py-2 text-right text-xs font-medium">Debit</th>
                  <th className="px-3 py-2 text-right text-xs font-medium">Credit</th>
                  <th className="px-3 py-2 text-right text-xs font-medium">Net Change</th>
                </tr>
              </thead>
              <tbody>
                {balances.map((balance) => (
                  <tr
                    key={balance.glAccountId}
                    className="hover:bg-muted/50 border-t transition-colors"
                  >
                    <td className="px-3 py-2 font-mono text-xs">{balance.accountCode}</td>
                    <td className="px-3 py-2 text-xs">{balance.accountName}</td>
                    <td className="text-muted-foreground px-3 py-2 text-xs capitalize">
                      {balance.accountCategory}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <AmountDisplay value={balance.periodDebitMinor} className="text-xs" />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <AmountDisplay value={balance.periodCreditMinor} className="text-xs" />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <AmountDisplay
                        value={balance.netChangeMinor}
                        variant="auto"
                        className="text-xs"
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot className="bg-muted/30 border-t font-medium">
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
                  <td className="px-3 py-2 text-right">
                    <AmountDisplay
                      value={totalNet}
                      variant="auto"
                      className="text-xs font-semibold"
                    />
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        )}
      </div>
    </PageLayout>
  );
}
