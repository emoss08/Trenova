import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** A page of the document at the width the viewer draws it at its starting zoom. */
export function DocumentPageSkeleton() {
  return <Skeleton className="aspect-[17/22] w-full max-w-153 rounded-sm" />;
}

/**
 * The rate confirmation while the viewer's code or the document's address is read: the
 * toolbar with the file name and the zoom controls over the first page, in the panel's own
 * tint, so the document lands inside its outline.
 */
export function DocumentPreviewPanelSkeleton() {
  const t = useT();

  return (
    <div
      className="bg-muted/20 flex h-full flex-col"
      aria-busy
      aria-label={t("Loading the document")}
    >
      <div className="contents" aria-hidden>
        <div className="bg-background/80 flex shrink-0 items-center justify-between border-b px-3 py-1.5">
          <div className="text-xs">
            <Skeleton className="inline-block h-2.5 w-36 align-middle" />
          </div>
          <div className="flex items-center gap-1">
            <Skeleton className="size-5.5" />
            <span className="flex w-10 justify-center">
              <Skeleton className="h-2 w-6" />
            </span>
            <Skeleton className="size-5.5" />
          </div>
        </div>
        <div className="flex min-h-0 flex-1 flex-col items-center gap-3 overflow-hidden p-4">
          <DocumentPageSkeleton />
        </div>
      </div>
    </div>
  );
}
