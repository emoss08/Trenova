import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/**
 * Stand-in for the palette while its chunk streams in. It mirrors the real
 * geometry — the search row, the scope strip, the list beside the preview and
 * the key strip — so nothing jumps when the palette takes over.
 */
export function CommandPaletteSkeleton() {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center px-4 pt-[12vh]">
      <div className="bg-scrim-subtle fixed inset-0" />
      <div className="bg-overlay ui-lift-float rounded-surface relative z-10 flex h-[min(40rem,calc(100dvh-16vh))] w-full max-w-4xl flex-col overflow-hidden">
        <div className="flex h-14 items-center gap-2.5 px-4">
          <Skeleton className="rounded-control size-4 shrink-0" />
          <Skeleton className="h-3.5 w-72" />
        </div>
        <div className="border-border-subtle flex h-9 items-center gap-4 border-b px-5">
          {Array.from({ length: 7 }, (_, tab) => (
            <Skeleton key={tab} className="h-2.5 w-14" />
          ))}
        </div>
        <div className="grid min-h-0 flex-1 md:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
          <div className="flex flex-col gap-4 p-4">
            {Array.from({ length: 3 }, (_, group) => (
              <div key={group} className="flex flex-col gap-2">
                <Skeleton className="h-2.5 w-24" />
                {Array.from({ length: 3 }, (_, row) => (
                  <div key={row} className="flex items-center gap-3">
                    <Skeleton className="rounded-control size-7 shrink-0" />
                    <div className="flex flex-1 flex-col gap-1.5">
                      <Skeleton className="h-3 w-2/5" />
                      <Skeleton className="h-2.5 w-3/5" />
                    </div>
                  </div>
                ))}
              </div>
            ))}
          </div>
          <div className="border-border-subtle bg-card hidden border-l md:block" />
        </div>
        <div className="border-border-subtle bg-sunken flex h-10 items-center gap-4 border-t px-4">
          <Skeleton className="h-2.5 w-20" />
          <Skeleton className="h-2.5 w-16" />
          <Skeleton className="h-2.5 w-16" />
        </div>
      </div>
    </div>
  );
}
