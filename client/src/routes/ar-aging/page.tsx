import { AmountDisplay } from "@/components/accounting/amount-display";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";

export function ARAgingPage() {
  const { data, isLoading } = useQuery({
    ...queries.ar.aging(),
  });

  const totals = data?.totals;
  const rows = data?.rows ?? [];

  return (
    <PageLayout
      pageHeaderProps={{
        title: "AR Aging",
        description: "Accounts receivable aging summary by customer.",
      }}
    >
      <div className="mx-4 mt-3 mb-4 space-y-4">
        {isLoading ? (
          <>
            <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-3 xl:grid-cols-6">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-20 w-full rounded-md" />
              ))}
            </div>
            <Skeleton className="h-64 w-full" />
          </>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-3 xl:grid-cols-6">
              <AgingKPICard label="Current" value={totals?.currentMinor ?? 0} />
              <AgingKPICard label="1-30 Days" value={totals?.days1To30Minor ?? 0} />
              <AgingKPICard label="31-60 Days" value={totals?.days31To60Minor ?? 0} />
              <AgingKPICard label="61-90 Days" value={totals?.days61To90Minor ?? 0} />
              <AgingKPICard label="90+ Days" value={totals?.daysOver90Minor ?? 0} variant="danger" />
              <AgingKPICard label="Total Open" value={totals?.totalOpenMinor ?? 0} variant="primary" />
            </div>

            <div className="overflow-hidden rounded-md border">
              <table className="w-full text-sm">
                <thead className="bg-muted/50 text-left text-muted-foreground">
                  <tr>
                    <th className="px-3 py-2 text-xs font-medium">Customer</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">Current</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">1-30</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">31-60</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">61-90</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">90+</th>
                    <th className="px-3 py-2 text-right text-xs font-medium">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr
                      key={row.customerId}
                      className="border-t transition-colors hover:bg-muted/50"
                    >
                      <td className="px-3 py-2">
                        <Link
                          to={`/accounting/ar/customer-ledger?customerId=${row.customerId}`}
                          className="text-xs font-medium hover:underline"
                        >
                          {row.customerName}
                        </Link>
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay value={row.buckets.currentMinor} className="text-xs" />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay value={row.buckets.days1To30Minor} className="text-xs" />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay value={row.buckets.days31To60Minor} className="text-xs" />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay value={row.buckets.days61To90Minor} className="text-xs" />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay value={row.buckets.daysOver90Minor} className="text-xs" />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={row.buckets.totalOpenMinor}
                          className="text-xs font-semibold"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
                {totals ? (
                  <tfoot className="border-t bg-muted/30 font-medium">
                    <tr>
                      <td className="px-3 py-2 text-xs">Totals</td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.currentMinor}
                          className="text-xs font-semibold"
                        />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.days1To30Minor}
                          className="text-xs font-semibold"
                        />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.days31To60Minor}
                          className="text-xs font-semibold"
                        />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.days61To90Minor}
                          className="text-xs font-semibold"
                        />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.daysOver90Minor}
                          className="text-xs font-semibold"
                        />
                      </td>
                      <td className="px-3 py-2 text-right">
                        <AmountDisplay
                          value={totals.totalOpenMinor}
                          className="text-xs font-bold"
                        />
                      </td>
                    </tr>
                  </tfoot>
                ) : null}
              </table>
            </div>
          </>
        )}
      </div>
    </PageLayout>
  );
}

function AgingKPICard({
  label,
  value,
  variant,
}: {
  label: string;
  value: number;
  variant?: "danger" | "primary";
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
            variant === "danger"
              ? "text-red-600 dark:text-red-400"
              : variant === "primary"
                ? "text-foreground"
                : ""
          }`}
        >
          {formatCurrency(value / 100)}
        </p>
      </CardContent>
    </Card>
  );
}
