import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/**
 * Stand-in for the explorer while the operation catalog streams in: the 340px
 * list rail with its search box and filter chips on the left, the document
 * header and SDL block on the right — the same geometry the real panels take,
 * so nothing shifts when the catalog lands.
 */
export function ExplorerSkeleton() {
  return (
    <div className="flex h-full min-h-0">
      <div className="border-border flex w-[340px] shrink-0 flex-col gap-2 border-r p-2">
        <Skeleton className="h-8 w-full rounded-md" />
        <div className="flex gap-1">
          {Array.from({ length: 4 }, (_, index) => (
            <Skeleton key={index} className="h-6 w-16 rounded-md" />
          ))}
        </div>
        <div className="flex flex-col gap-1 pt-1">
          {Array.from({ length: 14 }, (_, index) => (
            <div key={index} className="flex items-center gap-2 px-1 py-1.5">
              <Skeleton className="h-3.5 w-10 shrink-0 rounded" />
              <Skeleton className="h-3 flex-1 rounded" />
            </div>
          ))}
        </div>
      </div>

      <div className="flex min-w-0 flex-1 flex-col gap-3 p-4">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-5 w-64 rounded" />
          <Skeleton className="h-3 w-80 rounded" />
        </div>
        <div className="flex gap-4">
          <Skeleton className="h-6 w-20 rounded" />
          <Skeleton className="h-6 w-20 rounded" />
          <Skeleton className="h-6 w-16 rounded" />
        </div>
        <Skeleton className="min-h-0 flex-1 rounded-md" />
      </div>
    </div>
  );
}
