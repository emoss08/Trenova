/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

export function SiteSearchLoading() {
  return (
    <div className="grow px-2 py-4">
      <div className="space-y-4">
        {/* Header Skeleton */}
        <div className="flex items-start justify-between">
          <div className="bg-muted h-4 w-full animate-pulse rounded" />
        </div>

        {/* Items Skeleton */}
        {[1, 2, 3].map((i) => (
          <div key={i} className="flex items-center space-x-2">
            <div className="border-muted-foreground/30 bg-muted-foreground/10 size-8 animate-pulse rounded-md border" />
            <div className="bg-muted h-4 w-full animate-pulse rounded" />
          </div>
        ))}

        {/* Second Section */}
        <div className="mt-6">
          <div className="bg-muted h-4 w-full animate-pulse rounded" />
          {[1, 2].map((i) => (
            <div key={i} className="mt-4 flex items-center space-x-2">
              <div className="border-muted-foreground/30 bg-muted-foreground/10 size-8 animate-pulse rounded-md border" />
              <div className="bg-muted h-4 w-full animate-pulse rounded" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
