import { Skeleton } from "@trenova/shared/components/ui/skeleton";

export function RoleBundlePickerSkeleton({ label }: { label: string }) {
  return (
    <div aria-busy aria-label={label} className="flex h-160 flex-col gap-6 overflow-hidden">
      <div aria-hidden className="flex items-center justify-between">
        <Skeleton className="h-7 w-full max-w-xs" />
        <Skeleton className="h-3 w-20" />
      </div>
      {Array.from({ length: 2 }, (_, section) => (
        <div key={section} aria-hidden>
          <div className="mb-2 flex items-center gap-2 px-1">
            <Skeleton className="h-3.5 w-20" />
            <Skeleton className="h-3 w-16" />
          </div>
          <div className="divide-border-subtle divide-y rounded-lg border">
            {Array.from({ length: 3 }, (__, row) => (
              <div key={row} className="flex items-center gap-4 px-4 py-3">
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-3.5 w-36" />
                  <Skeleton className="h-3 w-2/3" />
                  <Skeleton className="h-3 w-16" />
                </div>
                <Skeleton className="h-4.5 w-8 rounded-full" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
