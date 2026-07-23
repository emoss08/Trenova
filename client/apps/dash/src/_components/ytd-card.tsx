import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { fetchMyYtdPay } from "@trenova/shared/lib/graphql/driver-portal";
import { useQuery } from "@tanstack/react-query";

export function YtdCard() {
  const year = new Date().getFullYear();
  const ytd = useQuery({
    queryKey: ["dash-ytd-pay", year],
    queryFn: async () => {
      try {
        return await fetchMyYtdPay(year);
      } catch {
        return null;
      }
    },
  });

  if (ytd.isPending) {
    return <Skeleton className="h-24 w-full rounded-2xl" />;
  }
  if (!ytd.data) {
    return null;
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <p className="text-2xs font-medium text-muted-foreground uppercase">{year} year to date</p>
      <p className="mt-1 text-2xl font-semibold tracking-tight">
        <AmountDisplay value={ytd.data.netPayMinor} />
      </p>
      <div className="mt-3 grid grid-cols-3 gap-2 border-t border-border pt-3 text-center">
        <div>
          <p className="text-2xs font-medium text-muted-foreground uppercase">Gross</p>
          <p className="mt-0.5 text-sm font-semibold tabular-nums">
            <AmountDisplay value={ytd.data.grossEarningsMinor} />
          </p>
        </div>
        <div>
          <p className="text-2xs font-medium text-muted-foreground uppercase">Deductions</p>
          <p className="mt-0.5 text-sm font-semibold tabular-nums">
            <AmountDisplay value={ytd.data.deductionsMinor} />
          </p>
        </div>
        <div>
          <p className="text-2xs font-medium text-muted-foreground uppercase">Statements</p>
          <p className="mt-0.5 text-sm font-semibold tabular-nums">{ytd.data.settlementCount}</p>
        </div>
      </div>
    </div>
  );
}
