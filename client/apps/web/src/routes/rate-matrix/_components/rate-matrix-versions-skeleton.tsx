import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The Versions tab while the matrix's versions load: the review state bar with its actions, then
 * rows of published versions.
 */
export function RateMatrixVersionsSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading versions")} className="flex flex-col gap-4">
      <div aria-hidden="true" className="flex items-center justify-between rounded-md border p-3">
        <div className="flex flex-col gap-1.5">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-64" />
        </div>
        <Skeleton className="h-8 w-32" />
      </div>
      <div aria-hidden="true" className="overflow-hidden rounded-md border">
        {[0, 1, 2].map((row) => (
          <div key={row} className="flex items-center gap-3 border-b px-3 py-2.5 last:border-b-0">
            <Skeleton className="h-3.5 w-10" />
            <Skeleton className="h-3.5 flex-1" />
            <Skeleton className="h-7 w-24" />
          </div>
        ))}
      </div>
    </div>
  );
}
