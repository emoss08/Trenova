import { AmountDisplay } from "@/components/accounting/amount-display";
import { SourceDrillDownLink } from "@/components/accounting/source-drill-down-link";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { SearchIcon } from "lucide-react";
import { useState } from "react";
import { useSearchParams } from "react-router";

export function CustomerLedgerPage() {
  const [searchParams] = useSearchParams();
  const initialCustomerId = searchParams.get("customerId") ?? "";
  const [customerId, setCustomerId] = useState(initialCustomerId);

  const { data, isLoading } = useQuery({
    ...queries.ar.customerLedger(customerId),
    enabled: Boolean(customerId),
  });

  const entries = data ?? [];

  let runningBalance = 0;
  const entriesWithBalance = entries.map((entry) => {
    runningBalance += entry.amountMinor;
    return { ...entry, runningBalance };
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Customer Ledger",
        description: "View transaction history for a customer.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        <div className="max-w-sm">
          <Input
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value)}
            placeholder="Enter customer ID..."
            leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          />
        </div>

        {!customerId ? (
          <div className="flex h-64 items-center justify-center rounded-lg border bg-card">
            <p className="text-sm text-muted-foreground">
              Enter a customer ID to view their ledger.
            </p>
          </div>
        ) : isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : entries.length === 0 ? (
          <div className="flex h-64 items-center justify-center rounded-lg border bg-card">
            <p className="text-sm text-muted-foreground">
              No ledger entries found for this customer.
            </p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-md border">
            <table className="w-full text-sm">
              <thead className="bg-muted/50 text-left text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 text-xs font-medium">Date</th>
                  <th className="px-3 py-2 text-xs font-medium">Document</th>
                  <th className="px-3 py-2 text-xs font-medium">Event Type</th>
                  <th className="px-3 py-2 text-xs font-medium">Source</th>
                  <th className="px-3 py-2 text-right text-xs font-medium">Amount</th>
                  <th className="px-3 py-2 text-right text-xs font-medium">Balance</th>
                </tr>
              </thead>
              <tbody>
                {entriesWithBalance.map((entry) => (
                  <tr
                    key={entry.id}
                    className="border-t transition-colors hover:bg-muted/50"
                  >
                    <td className="px-3 py-2 text-xs">
                      {new Date(entry.transactionDate * 1000).toLocaleDateString()}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">{entry.documentNumber}</td>
                    <td className="px-3 py-2 text-xs capitalize text-muted-foreground">
                      {entry.sourceEventType}
                    </td>
                    <td className="px-3 py-2">
                      <SourceDrillDownLink
                        sourceType={entry.sourceObjectType}
                        sourceId={entry.sourceObjectId}
                      />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <AmountDisplay value={entry.amountMinor} variant="auto" className="text-xs" />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <AmountDisplay
                        value={entry.runningBalance}
                        variant="auto"
                        className="text-xs font-medium"
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </PageLayout>
  );
}
