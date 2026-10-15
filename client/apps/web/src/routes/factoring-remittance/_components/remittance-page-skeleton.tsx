import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/** The remittance page as it loads: the totals strip, then the two line panels. */
export function RemittancePageSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy="true">
      <Skeleton className="h-16 w-full" />
      <div className="grid gap-4 xl:grid-cols-2">
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    </div>
  );
}
