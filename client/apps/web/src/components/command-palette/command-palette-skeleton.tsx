import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/**
 * Stand-in for the command dialog while its chunk streams in. It mirrors the
 * real palette's geometry — centred 640px card, 44px input row, grouped result
 * rows — so the dialog does not jump when the real component takes over.
 */
export function CommandPaletteSkeleton() {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      <div className="fixed inset-0 bg-black/40" />
      <div className="border-border bg-popover relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-lg border shadow-lg">
        <div className="border-border flex h-11 items-center gap-2 border-b px-3">
          <Skeleton className="size-4 shrink-0 rounded" />
          <Skeleton className="h-3.5 w-48 rounded" />
        </div>
        <div className="flex flex-col gap-3 p-2">
          {Array.from({ length: 2 }, (_, group) => (
            <div key={group} className="flex flex-col gap-1.5">
              <Skeleton className="mx-2 h-2.5 w-20 rounded" />
              {Array.from({ length: 4 }, (_, row) => (
                <div key={row} className="flex items-center gap-2 px-2 py-1.5">
                  <Skeleton className="size-4 shrink-0 rounded" />
                  <Skeleton className="h-3 w-full max-w-64 rounded" />
                </div>
              ))}
            </div>
          ))}
        </div>
        <div className="border-border flex h-9 items-center gap-3 border-t px-3">
          <Skeleton className="h-2.5 w-16 rounded" />
          <Skeleton className="h-2.5 w-16 rounded" />
          <Skeleton className="ml-auto h-2.5 w-24 rounded" />
        </div>
      </div>
    </div>
  );
}
