import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

function SkeletonLine({ className }: { className?: string }) {
  return <Skeleton className={cn("h-3 rounded-sm", className)} />;
}

function SkeletonToolbar({ count = 3 }: { count?: number }) {
  return (
    <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
      <div className="space-y-2">
        <SkeletonLine className="w-40" />
        <SkeletonLine className="w-56" />
      </div>
      <div className="flex shrink-0 gap-2">
        {Array.from({ length: count }).map((_, index) => (
          <Skeleton key={index} className="h-8 w-24 rounded-md" />
        ))}
      </div>
    </div>
  );
}

export function DesignerPanelSkeleton() {
  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <SkeletonToolbar count={2} />
      <div className="grid min-h-0 grid-cols-[minmax(0,1fr)_320px]">
        <div className="space-y-3 p-3">
          {Array.from({ length: 8 }).map((_, index) => (
            <Skeleton key={index} className="h-9 rounded-md" />
          ))}
        </div>
        <div className="space-y-3 border-l p-3">
          <SkeletonLine className="w-32" />
          {Array.from({ length: 7 }).map((_, index) => (
            <Skeleton key={index} className="h-10 rounded-md" />
          ))}
        </div>
      </div>
    </div>
  );
}

export function DesignerAsideSkeleton() {
  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-md border bg-background">
      <div className="flex h-11 items-center gap-2 border-b px-3">
        <Skeleton className="size-4 rounded-sm" />
        <SkeletonLine className="w-24" />
      </div>
      <div className="space-y-3 border-b p-3">
        <Skeleton className="h-8 rounded-md" />
        <Skeleton className="h-8 rounded-md" />
        <div className="grid grid-cols-2 gap-2">
          <Skeleton className="h-8 rounded-md" />
          <Skeleton className="h-8 rounded-md" />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {Array.from({ length: 8 }).map((_, index) => (
          <div key={index} className="space-y-2 border-b px-3 py-3">
            <SkeletonLine className="w-4/5" />
            <SkeletonLine className="w-2/3" />
          </div>
        ))}
      </div>
      <div className="space-y-2 border-t p-3">
        <SkeletonLine className="w-36" />
        <Skeleton className="h-8 rounded-md" />
        <Skeleton className="h-8 rounded-md" />
      </div>
    </aside>
  );
}

export function DesignerWorkspaceSkeleton() {
  return (
    <div
      className="grid h-[calc(100vh-11rem)] min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-3 overflow-hidden"
      aria-label="Loading EDI designer workspace"
    >
      <div className="flex w-fit gap-1 rounded-md bg-muted p-1">
        <Skeleton className="h-8 w-28 rounded-sm" />
        <Skeleton className="h-8 w-52 rounded-sm" />
      </div>
      <div className="grid min-h-0 grid-cols-[310px_minmax(0,1fr)_380px] gap-3 overflow-hidden max-xl:grid-cols-[280px_minmax(0,1fr)] max-lg:grid-cols-1">
        <DesignerAsideSkeleton />
        <main className="flex min-h-0 flex-col overflow-hidden rounded-md border bg-background">
          <SkeletonToolbar count={4} />
          <div className="grid min-h-0 flex-1 grid-cols-[280px_minmax(0,1fr)] max-md:grid-cols-1">
            <div className="min-h-0 border-r max-md:hidden">
              <div className="space-y-2 border-b p-3">
                <SkeletonLine className="w-24" />
                <Skeleton className="h-8 rounded-md" />
              </div>
              {Array.from({ length: 10 }).map((_, index) => (
                <div key={index} className="space-y-2 border-b px-3 py-2">
                  <SkeletonLine className="w-32" />
                  <SkeletonLine className="w-44" />
                </div>
              ))}
            </div>
            <DesignerPanelSkeleton />
          </div>
          <div className="flex items-center justify-between border-t px-3 py-2">
            <SkeletonLine className="w-96 max-w-[55%]" />
            <Skeleton className="h-8 w-32 rounded-md" />
          </div>
        </main>
        <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-md border bg-background max-xl:hidden">
          <div className="flex h-11 items-center gap-2 border-b px-3">
            <Skeleton className="size-4 rounded-sm" />
            <SkeletonLine className="w-24" />
          </div>
          <div className="space-y-3 p-3">
            {Array.from({ length: 6 }).map((_, index) => (
              <div key={index} className="space-y-2 rounded-md border p-2">
                <SkeletonLine className="w-20" />
                <SkeletonLine className="w-full" />
              </div>
            ))}
          </div>
        </aside>
      </div>
    </div>
  );
}
