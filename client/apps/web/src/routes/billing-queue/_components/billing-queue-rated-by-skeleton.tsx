import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const RULE_ROWS = ["w-28", "w-36", "w-24", "w-20"] as const;
const ACCESSORIAL_ROWS = ["w-48", "w-40", "w-44"] as const;

/**
 * The "Rated by" card while its terms load, drawn as the card lays out: the agreement's
 * name and dates, the rule that matched, its accessorials, the fuel program and the line
 * saying when it was rated.
 */
export function BillingQueueRatedBySkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col" aria-busy aria-label={t("Loading the terms")}>
      <div className="contents" aria-hidden>
        <div className="border-border flex flex-col gap-1.5 border-b px-3.5 py-3">
          <div className="flex items-center gap-2">
            <Skeleton className="h-3.5 w-40" />
            <Skeleton className="h-4 w-12" />
          </div>
          <Skeleton className="h-2.5 w-56" />
        </div>
        <div className="flex flex-col gap-3 px-3.5 py-3">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-2.5 w-28" />
            {RULE_ROWS.map((width, index) => (
              <div key={index} className="grid grid-cols-[6rem_minmax(0,1fr)] gap-3">
                <Skeleton className="h-3 w-16" />
                <Skeleton className={`h-3 ${width}`} />
              </div>
            ))}
          </div>
          <div className="flex flex-col gap-2">
            <Skeleton className="h-2.5 w-36" />
            {ACCESSORIAL_ROWS.map((width, index) => (
              <div key={index} className="flex items-center justify-between gap-2">
                <Skeleton className={`h-3 ${width}`} />
                <Skeleton className="h-4 w-14" />
              </div>
            ))}
          </div>
          <div className="flex flex-col gap-1.5">
            <Skeleton className="h-2.5 w-8" />
            <Skeleton className="h-3 w-60" />
          </div>
        </div>
        <div className="border-border flex items-center justify-between gap-2 border-t px-3.5 py-2.5">
          <Skeleton className="h-2.5 w-36" />
          <Skeleton className="h-3 w-24" />
        </div>
      </div>
    </div>
  );
}
