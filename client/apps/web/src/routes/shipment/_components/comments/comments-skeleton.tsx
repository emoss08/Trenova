import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/**
 * Stand-in for the comments tab: filter toolbar, a stream of avatar + body
 * rows, and the composer box pinned to the bottom. Deliberately free of any
 * import from the tab itself so it can be the Suspense fallback without
 * dragging the tab's chunk back in.
 */
export function CommentsTabSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 px-4 py-2">
        <Skeleton className="h-7 flex-1 rounded-md" />
        <Skeleton className="size-7 shrink-0 rounded-md" />
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-4 px-4 py-3">
        {Array.from({ length: 5 }, (_, index) => (
          <div key={index} className="flex gap-2.5">
            <Skeleton className="size-7 shrink-0 rounded-full" />
            <div className="flex flex-1 flex-col gap-1.5">
              <Skeleton className="h-3 w-40 rounded" />
              <Skeleton className="h-3 w-full rounded" />
              <Skeleton className="h-3 w-3/4 rounded" />
            </div>
          </div>
        ))}
      </div>
      <div className="border-border shrink-0 border-t px-4 pt-3 pb-4">
        <Skeleton className="h-[132px] w-full rounded-md" />
      </div>
    </div>
  );
}
