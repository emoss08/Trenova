import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  fetchMyPtoBalances,
  type PortalPtoBalance,
} from "@trenova/shared/lib/graphql/driver-portal";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ptoTypeLabels } from "./portal-badges";

function days(value: string): string {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed.toFixed(parsed % 1 === 0 ? 0 : 2) : value;
}

export function useMyPtoBalances(enabled: boolean) {
  return useQuery({
    queryKey: ["dash-pto-balances"],
    queryFn: ({ signal }) => fetchMyPtoBalances({ signal }),
    enabled,
    staleTime: 60 * 1000,
  });
}

export function PtoBalanceStrip({
  balances,
  isPending,
}: {
  balances: PortalPtoBalance[] | undefined;
  isPending: boolean;
}) {
  if (isPending) {
    return <Skeleton className="mb-3 h-14 w-full rounded-xl" />;
  }
  if (!balances || balances.length === 0) {
    return null;
  }

  return (
    <div
      className="mb-3 grid gap-2"
      style={{ gridTemplateColumns: `repeat(${Math.min(balances.length, 3)}, minmax(0, 1fr))` }}
      data-testid="pto-balance-strip"
    >
      {balances.map((balance) => {
        const available = Number(balance.availableDays);
        return (
          <div
            key={balance.ptoType}
            className="border-border bg-muted/30 rounded-xl border px-3 py-2"
          >
            <p className="text-2xs text-muted-foreground uppercase">
              {ptoTypeLabels[balance.ptoType] ?? balance.ptoType}
            </p>
            <p
              className={cn(
                "text-lg font-semibold tabular-nums",
                available < 0 && "text-destructive",
              )}
            >
              {days(balance.availableDays)}
              <span className="text-2xs text-muted-foreground ml-1 font-normal">days</span>
            </p>
            <p className="text-2xs text-muted-foreground">
              {Number(balance.pendingDays) > 0
                ? `${days(balance.pendingDays)} pending`
                : "available"}
              {balance.nextAccrual
                ? ` · +${days(balance.nextAccrual.nominalDays)} ${formatUnixDateMedium(balance.nextAccrual.effectiveAt)}`
                : ""}
            </p>
          </div>
        );
      })}
    </div>
  );
}
