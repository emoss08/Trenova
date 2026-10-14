import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The tracking page's layout while it loads: the top bar, the headline and
 * arrival, the route line, then the stops beside the side column.
 */
export function TrackingViewSkeleton() {
  const t = useT();

  return (
    <div className="relative min-h-full" aria-busy="true" aria-label={t("Loading the shipment")}>
      <div aria-hidden className="mx-auto flex w-full max-w-6xl flex-col px-5 md:px-8">
        <div className="flex h-16 items-center gap-2.5">
          <Skeleton className="size-6 rounded-full" />
          <Skeleton className="h-4 w-32" />
        </div>
        <div className="grid gap-10 pt-10 pb-12 md:pt-16 lg:grid-cols-[minmax(0,1fr)_minmax(0,400px)] lg:items-end lg:gap-16">
          <div className="flex flex-col gap-6">
            <Skeleton className="h-5 w-48" />
            <Skeleton className="h-16 w-full max-w-xl" />
          </div>
          <div className="flex flex-col gap-3">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-9 w-56" />
            <Skeleton className="h-5 w-64" />
          </div>
        </div>
        <div className="border-y border-border-subtle py-8">
          <Skeleton className="h-36 w-full" />
        </div>
        <div className="grid gap-12 py-12 lg:grid-cols-[minmax(0,1fr)_360px] lg:gap-16">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    </div>
  );
}
