import { useT } from "@trenova/shared/i18n/use-t";
import { SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";

const ROWS = ["w-3/5", "w-2/5", "w-1/2", "w-3/5", "w-2/5", "w-1/2"];

/**
 * Placeholder rows shaped like a notification: the 28px tile, a title line, two lines of
 * message and the row's action. Shown while a part of the panel loads and while the
 * panel's own code does.
 */
export function NotificationFeedSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col px-1.5" aria-busy aria-label={t("Loading notifications")}>
      <Skeleton className="mx-2.5 mt-3.5 mb-1.5 h-2.5 w-10 rounded-md" />
      {ROWS.map((width, index) => (
        <div key={index} className="grid grid-cols-[1.75rem_minmax(0,1fr)] gap-x-2.5 p-2.5">
          <Skeleton className="mt-px size-7 rounded-md" />
          <div className="flex flex-col gap-1.5 pt-0.5">
            <Skeleton className={`h-3 rounded-md ${width}`} />
            <Skeleton className="h-2.5 w-11/12 rounded-md" />
            <Skeleton className="h-2.5 w-2/3 rounded-md" />
            <Skeleton className="mt-1 h-6 w-24 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * The whole panel while its code loads: the title, the line that says what waits, the tab
 * strip and the feed, so the sheet does not reflow when the panel takes over. It keeps a
 * real SheetTitle so the dialog stays labelled meanwhile.
 */
export function NotificationPanelSkeleton() {
  const t = useT();

  return (
    <>
      <div className="flex flex-col gap-2.5 pt-3 pr-3 pl-4">
        <div className="flex min-h-7 items-center pr-9">
          <SheetTitle className="text-lg font-semibold">{t("Notifications")}</SheetTitle>
        </div>
        <Skeleton className="-mt-1 h-3 w-40 rounded-md" />
        <div className="-mr-3 -ml-4 flex h-8 items-center gap-4 border-b border-border pr-3 pl-4">
          <Skeleton className="h-3 w-14 rounded-md" />
          <Skeleton className="h-3 w-12 rounded-md" />
          <Skeleton className="h-3 w-10 rounded-md" />
          <Skeleton className="h-3 w-9 rounded-md" />
        </div>
      </div>
      <NotificationFeedSkeleton />
    </>
  );
}
