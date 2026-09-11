import { useT } from "@trenova/shared/i18n/use-t";
import { SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";

/**
 * Placeholder rows shaped like a notification: a 28px square tile followed by a
 * title line and a shorter body line. Used while the feed query is in flight
 * and while the panel chunk is loading.
 */
export function NotificationFeedSkeleton() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 6 }, (_, index) => (
        <div key={index} className="flex gap-3 px-4 py-3">
          <Skeleton className="size-7 shrink-0 rounded-md" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-3/5 rounded" />
            <Skeleton className="h-2.5 w-4/5 rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * Full-panel placeholder: the header row, the Inbox/Archive tab strip and the
 * feed, so the sheet does not reflow when the real panel takes over. It keeps a
 * real SheetTitle so the dialog stays labelled while the chunk loads.
 */
export function NotificationPanelSkeleton() {
  const t = useT();

  return (
    <>
      <div className="flex items-center justify-between gap-2 py-3 pr-11 pl-4">
        <SheetTitle className="text-sm font-semibold">{t("Notifications")}</SheetTitle>
      </div>
      <div className="border-border flex items-center gap-3 border-b pr-3 pl-4">
        <Skeleton className="my-2 h-4 w-12 rounded" />
        <Skeleton className="my-2 h-4 w-14 rounded" />
      </div>
      <NotificationFeedSkeleton />
    </>
  );
}
