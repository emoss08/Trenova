import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/** A history of an extension's runs while it loads: its heading and four rows. */
export function RunHistorySkeleton({ label }: { label: string }) {
  return (
    <div aria-busy aria-label={label} className="flex flex-col gap-3">
      <div aria-hidden className="contents">
        <Skeleton className="h-4 w-32" />
        <div className="flex flex-col divide-y rounded-md border">
          {[0, 1, 2, 3].map((row) => (
            <div key={row} className="flex items-center gap-3 px-3 py-2.5">
              <Skeleton className="h-5 w-16" />
              <Skeleton className="h-4 flex-1" />
              <Skeleton className="h-3 w-20" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
