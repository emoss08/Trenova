import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The log section as it lands: its title, the sentence, the period and button, and two requests. */
export function SendLogsSkeleton() {
  const t = useT();

  return (
    <div
      className="flex flex-col overflow-hidden rounded-lg border bg-card"
      aria-busy
      aria-label={t("Loading log requests")}
    >
      <div aria-hidden className="flex flex-col">
        <div className="flex min-h-9 items-center border-b border-border px-3 py-1.5">
          <Skeleton className="h-4 w-52" />
        </div>
        <div className="flex flex-col gap-3 px-3 py-3">
          <Skeleton className="h-3 w-full max-w-md" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-7 w-36" />
            <Skeleton className="h-7 w-24" />
          </div>
        </div>
        <div className="flex flex-col divide-y divide-border-subtle border-t border-border-subtle">
          {Array.from({ length: 2 }, (_, index) => (
            <div
              key={index}
              className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 px-3 py-2.5"
            >
              <Skeleton className="h-3.5 w-24" />
              <Skeleton className="col-start-2 row-span-2 row-start-1 h-5 w-20" />
              <Skeleton className="col-start-1 h-3 w-56" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
