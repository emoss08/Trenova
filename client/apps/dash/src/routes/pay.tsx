import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { DriverSettlementStatusBadge } from "@trenova/shared/components/status-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatRange } from "@trenova/shared/lib/date";
import { fetchMySettlements } from "@trenova/shared/lib/graphql/driver-portal";
import type { DriverSettlementStatus } from "@trenova/shared/types/driver-pay";
import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon, ReceiptTextIcon } from "lucide-react";
import { m } from "motion/react";
import { Link } from "react-router";
import { YtdCard } from "../_components/ytd-card";

export function DashPayPage() {
  const settlements = useQuery({
    queryKey: ["dash-settlements"],
    queryFn: () => fetchMySettlements(50, 0),
  });

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Pay</h1>
        <p className="text-sm text-muted-foreground">Your settlement statements, newest first.</p>
      </div>

      <YtdCard />

      {settlements.isPending ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-20 w-full rounded-2xl" />
          <Skeleton className="h-20 w-full rounded-2xl" />
          <Skeleton className="h-20 w-full rounded-2xl" />
        </div>
      ) : settlements.data && settlements.data.items.length > 0 ? (
        <ul className="flex flex-col gap-3">
          {settlements.data.items.map((settlement, index) => (
            <m.li
              key={settlement.id}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{
                duration: 0.22,
                ease: "easeOut",
                delay: Math.min(index * 0.04, 0.24),
              }}
            >
              <Link to={`/dash/pay/${settlement.id}`} className="block">
                <m.div
                  whileTap={{ scale: 0.98 }}
                  className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-card p-4 transition-colors hover:border-foreground/20"
                >
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="truncate font-mono text-sm font-semibold">
                        {settlement.settlementNumber}
                      </p>
                      <DriverSettlementStatusBadge
                        status={settlement.status as DriverSettlementStatus}
                      />
                    </div>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {formatRange(settlement.periodStart, settlement.periodEnd)} · Pay date{" "}
                      {formatRange(settlement.payDate, settlement.payDate)}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <span className="text-sm font-semibold tabular-nums">
                      <AmountDisplay
                        value={settlement.netPayMinor}
                        currency={settlement.currencyCode}
                      />
                    </span>
                    <ChevronRightIcon className="size-4 text-muted-foreground" />
                  </div>
                </m.div>
              </Link>
            </m.li>
          ))}
        </ul>
      ) : (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-10 text-center">
          <ReceiptTextIcon className="size-6 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            Once your carrier issues a settlement, your statement will show up here.
          </p>
        </div>
      )}
    </div>
  );
}
