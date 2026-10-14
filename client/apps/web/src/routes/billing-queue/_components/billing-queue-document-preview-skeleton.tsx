import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const MODE_WIDTHS = ["w-9", "w-9", "w-8"] as const;

/**
 * A supporting document while its viewer's code or its address is read, drawn as the PDF viewer
 * draws it in scroll mode: the toolbar with the mode switch and the zoom controls over the
 * first page at the height the viewer gives a page that is still rendering. Billing queue and
 * invoices open the same preview, so both draw this.
 */
export function BillingQueueDocumentPreviewSkeleton() {
  const t = useT();

  return (
    <div
      className="bg-background flex h-full flex-col overflow-hidden"
      aria-busy
      aria-label={t("Loading the document")}
    >
      <div className="contents" aria-hidden>
        <div className="bg-muted/50 flex items-center justify-between gap-4 border-b p-3">
          <div className="bg-background flex items-center gap-1 rounded-md border p-1">
            {MODE_WIDTHS.map((width, index) => (
              <span key={index} className="flex h-7 items-center px-3">
                <Skeleton className={`h-2.5 ${width}`} />
              </span>
            ))}
          </div>
          <div className="flex items-center gap-2">
            <Skeleton className="h-7 w-7" />
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-7 w-7" />
            <Skeleton className="h-7 w-10" />
          </div>
        </div>
        <div className="bg-muted/30 min-h-0 flex-1 overflow-hidden p-4">
          <Skeleton className="h-200 w-full" />
        </div>
      </div>
    </div>
  );
}
