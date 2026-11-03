import { pageAspect } from "@/lib/document-reading/page-geometry";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const RAIL_THUMBS = 5;

/**
 * The packet viewer while the packet is assembled, drawn as the viewer draws it: the rail of
 * thumbnails under the first document's name beside a column of letter pages, with the toolbar
 * over its foot.
 */
export function BillingQueueDocumentPagesSkeleton() {
  const t = useT();

  return (
    <div
      className="bg-muted flex h-full min-h-0 overflow-hidden"
      aria-busy
      aria-label={t("Loading the document")}
    >
      <div className="contents" aria-hidden>
        <div className="border-border bg-card hidden w-38 shrink-0 flex-col items-center gap-3 border-r px-4 pt-3 sm:flex">
          <div className="flex w-full items-center justify-between">
            <Skeleton className="h-2.5 w-16" />
            <Skeleton className="h-2.5 w-4" />
          </div>
          {Array.from({ length: RAIL_THUMBS }, (_, index) => (
            <div key={index} className="flex w-16 flex-col items-center gap-1.5">
              <Skeleton className="w-full rounded-xs" style={{ aspectRatio: pageAspect(null) }} />
              <Skeleton className="h-2 w-3" />
            </div>
          ))}
        </div>
        <div className="relative flex min-w-0 flex-1 flex-col items-center gap-4 overflow-hidden px-4 pt-6">
          {[0, 1].map((index) => (
            <div
              key={index}
              className="border-border bg-card relative w-[min(100%,520px)] shrink-0 border"
              style={{ aspectRatio: pageAspect(null) }}
            >
              <Skeleton className="absolute inset-0 rounded-none" />
            </div>
          ))}
          <Skeleton className="absolute bottom-3 left-1/2 h-8 w-80 -translate-x-1/2 rounded-lg" />
        </div>
      </div>
    </div>
  );
}
