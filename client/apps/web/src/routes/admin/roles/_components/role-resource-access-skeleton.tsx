import { Skeleton } from "@trenova/shared/components/ui/skeleton";

export function RoleResourceAccessSkeleton({ label }: { label: string }) {
  return (
    <div
      aria-busy
      aria-label={label}
      className="bg-card flex flex-col overflow-hidden rounded-lg border md:h-160 md:flex-row"
    >
      <div
        aria-hidden
        className="flex flex-col gap-3 border-b p-2 md:w-56 md:border-r md:border-b-0"
      >
        <Skeleton className="h-7 w-full" />
        <div className="hidden flex-col gap-2 px-2 md:flex">
          {Array.from({ length: 12 }, (_, item) => (
            <div key={item} className="flex items-center justify-between">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-3 w-6" />
            </div>
          ))}
        </div>
      </div>
      <div aria-hidden className="flex min-w-0 flex-1 flex-col">
        <div className="flex h-10 items-center justify-between border-b px-4">
          <Skeleton className="h-3.5 w-32" />
          <Skeleton className="h-6 w-16" />
        </div>
        {Array.from({ length: 8 }, (_, row) => (
          <div
            key={row}
            className="border-border-subtle flex items-center gap-3 border-b px-4 py-2.5"
          >
            <div className="flex-1 space-y-1.5">
              <Skeleton className="h-3.5 w-40" />
              <Skeleton className="h-3 w-64" />
            </div>
            <Skeleton className="h-7 w-32" />
          </div>
        ))}
      </div>
    </div>
  );
}
