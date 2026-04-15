import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ArrowRightIcon } from "lucide-react";
import { Link } from "react-router";

export function ReconciliationSummaryPage() {
  const { data, isLoading } = useQuery({
    ...queries.bankReceipt.summary(),
  });

  const matchRate =
    data && data.importedCount > 0
      ? Math.round((data.matchedCount / data.importedCount) * 100)
      : 0;

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Reconciliation Summary",
        description: "Overview of bank receipt reconciliation status.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-6">
        {isLoading ? (
          <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 w-full rounded-md" />
            ))}
          </div>
        ) : data ? (
          <>
            <div className="grid grid-cols-2 gap-2.5 xl:grid-cols-4">
              <SummaryKPICard
                label="Imported"
                count={data.importedCount}
                amount={data.importedAmount}
              />
              <SummaryKPICard
                label="Matched"
                count={data.matchedCount}
                amount={data.matchedAmount}
              />
              <SummaryKPICard
                label="Exceptions"
                count={data.exceptionCount}
                amount={data.exceptionAmount}
                variant="danger"
              />
              <Card className="gap-0 overflow-hidden rounded-md">
                <CardHeader className="pb-1">
                  <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
                    Match Rate
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-3xl font-semibold tabular-nums tracking-tight">
                    {matchRate}%
                  </p>
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-4 xl:grid-cols-2">
              <Card className="rounded-md">
                <CardHeader>
                  <CardTitle className="text-sm font-semibold">Exception Aging</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="overflow-hidden rounded-md border">
                    <table className="w-full text-sm">
                      <thead className="bg-muted/50 text-left text-muted-foreground">
                        <tr>
                          <th className="px-3 py-2 text-xs font-medium">Period</th>
                          <th className="px-3 py-2 text-right text-xs font-medium">Count</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr className="border-t">
                          <td className="px-3 py-2 text-xs">Current</td>
                          <td className="px-3 py-2 text-right font-mono text-xs">
                            {data.exceptionAging.currentCount}
                          </td>
                        </tr>
                        <tr className="border-t">
                          <td className="px-3 py-2 text-xs">1-3 Days</td>
                          <td className="px-3 py-2 text-right font-mono text-xs">
                            {data.exceptionAging.days1To3Count}
                          </td>
                        </tr>
                        <tr className="border-t">
                          <td className="px-3 py-2 text-xs">4-7 Days</td>
                          <td className="px-3 py-2 text-right font-mono text-xs">
                            {data.exceptionAging.days4To7Count}
                          </td>
                        </tr>
                        <tr className="border-t">
                          <td className="px-3 py-2 text-xs">7+ Days</td>
                          <td className="px-3 py-2 text-right font-mono text-xs font-semibold text-red-600 dark:text-red-400">
                            {data.exceptionAging.daysOver7Count}
                          </td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>

              <Card className="rounded-md">
                <CardHeader>
                  <CardTitle className="text-sm font-semibold">Work Items</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">Active</span>
                      <span className="font-mono font-medium">{data.activeWorkItemCount}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">Assigned</span>
                      <span className="font-mono font-medium">{data.assignedWorkItemCount}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">In Review</span>
                      <span className="font-mono font-medium">{data.inReviewWorkItemCount}</span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            <div className="flex items-center gap-3">
              <Link to="/accounting/reconciliation/bank-receipts">
                <Button variant="outline" size="sm">
                  Bank Receipts
                  <ArrowRightIcon className="ml-1.5 size-3.5" />
                </Button>
              </Link>
              <Link to="/accounting/reconciliation/work-queue">
                <Button variant="outline" size="sm">
                  Work Queue
                  <ArrowRightIcon className="ml-1.5 size-3.5" />
                </Button>
              </Link>
            </div>
          </>
        ) : null}
      </div>
    </PageLayout>
  );
}

function SummaryKPICard({
  label,
  count,
  amount,
  variant,
}: {
  label: string;
  count: number;
  amount: number;
  variant?: "danger";
}) {
  return (
    <Card className="gap-0 overflow-hidden rounded-md">
      <CardHeader className="pb-1">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          {label}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p
          className={`text-2xl font-semibold tabular-nums tracking-tight ${
            variant === "danger" ? "text-red-600 dark:text-red-400" : ""
          }`}
        >
          {count}
        </p>
        <p className="text-xs text-muted-foreground tabular-nums">
          {formatCurrency(amount / 100)}
        </p>
      </CardContent>
    </Card>
  );
}
