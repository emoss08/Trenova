import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Stands in twice: as the Suspense fallback while the notifications tab's
 * chunk arrives, and as the list's own loading state while the feed is read.
 *
 * Each row mirrors a loaded notification: the priority icon, the title, a
 * message line on the rows that carry one, and the source and operation
 * badges with the table name and the time. Each text bar sits in a box the
 * height of the line it stands for.
 */

const TITLE_WIDTHS = ["w-64", "w-52", "w-72", "w-48", "w-60", "w-56", "w-68", "w-44"] as const;
const MESSAGE_WIDTHS = ["w-96", "", "w-80", "w-88", "", "w-72", "", "w-96"] as const;
const TABLE_NAME_WIDTHS = ["w-16", "w-20", "w-14", "w-24", "w-16", "w-20", "w-14", "w-18"] as const;

export function NotificationListSkeleton() {
  const t = useT();

  return (
    <div aria-busy aria-label={t("Loading notifications")}>
      <div aria-hidden className="flex flex-col gap-1">
        {TITLE_WIDTHS.map((titleWidth, index) => {
          const messageWidth = MESSAGE_WIDTHS[index];
          return (
            <div
              key={index}
              className="flex items-start gap-3 rounded-lg border border-transparent px-4 py-3"
            >
              <Skeleton className="mt-0.5 size-4 shrink-0 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                <span className="flex h-4 items-center">
                  <Skeleton className={cn("h-3 max-w-full", titleWidth)} />
                </span>
                {messageWidth ? (
                  <span className="flex h-4 items-center">
                    <Skeleton className={cn("h-2 max-w-full", messageWidth)} />
                  </span>
                ) : null}
                <div className="flex flex-wrap items-center gap-1.5">
                  <Skeleton className="h-5 w-16 rounded-full" />
                  <Skeleton className="h-5 w-14 rounded-full" />
                  <Skeleton className={cn("h-2", TABLE_NAME_WIDTHS[index])} />
                  <Skeleton className="h-2 w-12" />
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
