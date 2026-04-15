import { AmountDisplay } from "@/components/accounting/amount-display";
import { FiscalPeriodSelector } from "@/components/accounting/fiscal-period-selector";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

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
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <FiscalPeriodSelector value={periodId} onChange={setPeriodId} />

        {!periodId ? (
          <div className="flex h-64 items-center justify-center rounded-lg border bg-card">
            <p className="text-sm text-muted-foreground">
              Select a fiscal period to view the trial balance.
            </p>
          </div>
        ) : isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-muted-foreground">
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
                    className="border-t transition-colors hover:bg-muted/50"
                  >
                    <td className="px-3 py-2 font-mono text-xs">{balance.accountCode}</td>
                    <td className="px-3 py-2 text-xs">{balance.accountName}</td>
                    <td className="px-3 py-2 text-xs capitalize text-muted-foreground">
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
