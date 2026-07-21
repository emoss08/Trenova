import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { formatCurrency } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { m } from "motion/react";
import { Link } from "react-router";

export function TopOverdueCustomersCard() {
  const { data: customers, isLoading } = useQuery(queries.ar.topOverdueCustomers(10));

  const rows = customers ?? [];
  const maxOverdue = rows.reduce((max, row) => Math.max(max, row.overdueMinor), 0);

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="flex flex-row items-center justify-between border-b py-3">
        <CardTitle className="text-sm font-medium">Top overdue customers</CardTitle>
        <Link
          to="/accounting/ar/aging"
          className="text-xs text-muted-foreground hover:text-foreground hover:underline"
        >
          View aging
        </Link>
      </CardHeader>
      <CardContent className="p-2">
        {isLoading ? (
          <div className="space-y-2 p-2">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-10 w-full" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex h-56 items-center justify-center text-sm text-muted-foreground">
            No overdue balances — nice work
          </div>
        ) : (
          <div className="max-h-80 divide-y overflow-y-auto">
            {rows.map((row, index) => (
              <m.div
                key={row.customerId}
                initial={{ opacity: 0, x: -6 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.25, delay: index * 0.03, ease: "easeOut" }}
              >
                <Link
                  to={`/accounting/ar/customer-ledger?customerId=${row.customerId}`}
                  className="flex items-center gap-3 rounded-md px-2 py-2 transition-colors hover:bg-muted/50"
                >
                  <span className="w-5 shrink-0 text-center text-xs font-medium text-muted-foreground tabular-nums">
                    {index + 1}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-baseline justify-between gap-2">
                      <span className="truncate text-xs font-medium">{row.customerName}</span>
                      <span className="shrink-0 text-xs font-semibold text-red-600 tabular-nums dark:text-red-400">
                        {formatCurrency(row.overdueMinor / 100)}
                      </span>
                    </div>
                    <div className="mt-1 flex items-center gap-2">
                      <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
                        <m.div
                          className="h-full rounded-full bg-red-500/70 dark:bg-red-400/70"
                          initial={{ width: 0 }}
                          animate={{
                            width:
                              maxOverdue > 0
                                ? `${(row.overdueMinor / maxOverdue) * 100}%`
                                : "0%",
                          }}
                          transition={{ duration: 0.5, delay: index * 0.03, ease: "easeOut" }}
                        />
                      </div>
                      <span className="shrink-0 text-[11px] text-muted-foreground tabular-nums">
                        {row.openInvoiceCount} inv · oldest {row.oldestDaysPastDue}d
                      </span>
                    </div>
                  </div>
                </Link>
              </m.div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
