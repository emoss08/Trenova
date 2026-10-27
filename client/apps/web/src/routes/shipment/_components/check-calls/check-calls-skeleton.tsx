import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** Two days of reports: each entry's title width, and whether it carries a meta line. */
const DAYS = [
  [
    [148, true],
    [196, true],
  ],
  [
    [172, false],
    [128, true],
    [184, true],
  ],
] as const;

export function CheckCallsSkeleton() {
  const t = useT();

  return (
    <div aria-busy="true" aria-label={t("Loading check calls")} className="flex flex-col gap-6">
      <div aria-hidden className="flex flex-col gap-4 rounded-lg border bg-card p-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-3 w-52" />
          </div>
          <Skeleton className="h-7 w-28" />
        </div>
        <div className="grid grid-cols-2 gap-3 border-t border-border-subtle pt-3">
          {[0, 1].map((figure) => (
            <div key={figure} className="flex flex-col gap-1.5">
              <Skeleton className="h-2.5 w-24" />
              <Skeleton className="h-3.5 w-20" />
            </div>
          ))}
        </div>
      </div>
      <div aria-hidden className="flex flex-col gap-5">
        {DAYS.map((entries, day) => (
          <div key={day} className="flex flex-col gap-3">
            <Skeleton className="h-3 w-16" />
            {entries.map(([width, meta], entry) => (
              <div key={entry} className="grid grid-cols-[3.5rem_1.5rem_minmax(0,1fr)] gap-x-3">
                <Skeleton className="mt-1 ml-auto h-2.5 w-10" />
                <Skeleton className="size-6 rounded-full" />
                <div className="flex flex-col gap-1.5 pt-0.5">
                  <Skeleton className="h-3.5" style={{ width }} />
                  {meta ? <Skeleton className="h-2.5 w-36" /> : null}
                </div>
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
