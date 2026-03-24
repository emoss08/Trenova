import { Skeleton } from "@/components/ui/skeleton";
import { TriangleAlert } from "lucide-react";

export function RequestedPTOOverviewSkeleton() {
  return (
    <div className="flex size-full flex-col gap-1 overflow-y-hidden rounded-md border border-border p-3">
      {Array.from({ length: 10 }).map((_, index) => (
        <div
          key={index}
          className="flex items-center justify-center rounded-md border border-border"
        >
          <Skeleton className="h-[70px] w-full" />
        </div>
      ))}
    </div>
  );
}
export function RequestedPTOEmptyState() {
  return (
    <div className="flex size-full flex-col items-center justify-center overflow-hidden rounded-md border border-border">
      <div className="relative size-full">
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1">
          <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
            No data available
          </p>
          <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
            Try adjusting your filters or search query
          </p>
        </div>
      </div>
    </div>
  );
}

export function RequestedPTOErrorState() {
  return (
    <div className="flex size-full flex-col items-center justify-center gap-1 overflow-hidden rounded-md border border-border p-3">
      <TriangleAlert className="mt-0.5 size-5 text-red-500" />
      <div className="flex flex-col items-center text-center">
        <p className="font-medium text-red-500">Error loading PTO requests</p>
        <p className="mt-1 text-xs text-muted-foreground">
          Looks like we hit a snag. Please try again later.
        </p>
      </div>
    </div>
  );
}
