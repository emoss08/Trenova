import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const ACTION_WIDTHS = ["w-56", "w-64", "w-48"] as const;

/**
 * The day's brief before it is read: two lines at the brief's own type size,
 * the row of next steps, and the "as of" line under them.
 */
export function AccountingBriefingSkeleton() {
  const t = useT();

  return (
    <div
      className="flex w-full min-w-0 flex-col gap-2"
      aria-busy
      aria-label={t("Loading the brief")}
    >
      <div className="contents" aria-hidden>
        <div className="flex flex-col gap-2 py-1 text-lg leading-relaxed">
          <Skeleton className="h-4 w-11/12" />
          <Skeleton className="h-4 w-3/5" />
        </div>
        <div className="flex flex-wrap gap-1.5">
          {ACTION_WIDTHS.map((width) => (
            <Skeleton key={width} className={cn("h-6 rounded-badge", width)} />
          ))}
        </div>
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-24 align-middle" />
        </div>
      </div>
    </div>
  );
}
