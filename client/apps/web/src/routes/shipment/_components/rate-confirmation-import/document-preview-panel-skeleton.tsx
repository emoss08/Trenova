import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The document's area while its file arrives: its first page, in the panel's own tint. */
export function DocumentBodySkeleton() {
  const t = useT();

  return (
    <div
      className="flex min-h-0 flex-1 flex-col items-center overflow-hidden p-4"
      aria-busy
      aria-label={t("Loading the document")}
    >
      <Skeleton className="aspect-[17/22] w-full max-w-153 rounded-sm" />
    </div>
  );
}

/**
 * The rate confirmation while the viewer's code arrives: the toolbar with the file name over
 * the first page, so the document lands inside its outline.
 */
export function DocumentPreviewPanelSkeleton() {
  return (
    <div className="bg-muted/20 flex h-full flex-col">
      <div className="bg-background/80 flex shrink-0 items-center border-b px-3 py-1.5" aria-hidden>
        <Skeleton className="h-2.5 w-36" />
      </div>
      <DocumentBodySkeleton />
    </div>
  );
}
