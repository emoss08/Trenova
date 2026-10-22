import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { AIActivityPanelSkeleton } from "./ai-activity-panel-skeleton";
import { DocumentPreviewPanelSkeleton } from "./document-preview-panel-skeleton";

/**
 * The reconciliation workspace while its code is read: the document, the review column and the
 * assistant side by side at the panels' starting widths, split by the same hairlines. The review
 * column keeps its header, the required details, the extracted fields grouped under their
 * sections and the bar that creates the shipment.
 */

const REQUIRED_FIELDS = 5;
const FIELD_SECTIONS = [
  { label: "w-16", fields: ["w-24", "w-32", "w-20", "w-28"] },
  { label: "w-12", fields: ["w-28", "w-16", "w-24"] },
] as const;
const COUNT_WIDTHS = ["w-16", "w-14", "w-12"] as const;

function ReviewLabelSkeleton({ width, first = false }: { width: string; first?: boolean }) {
  return (
    <div className={cn("px-3 pb-1 text-xs", first ? "pt-2" : "pt-3")}>
      <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
    </div>
  );
}

function ReviewColumnSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center justify-between gap-2 border-b px-4 py-2.5">
        <div className="flex items-center gap-3 text-xs">
          <Skeleton className="inline-block h-2.5 w-20 align-middle" />
          {COUNT_WIDTHS.map((width, index) => (
            <span key={index} className="flex items-center gap-1.5">
              <Skeleton className="size-1.5 rounded-full" />
              <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
            </span>
          ))}
        </div>
        <Skeleton className="h-7 w-28" />
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        <div className="border-b">
          <ReviewLabelSkeleton width="w-24" first />
          <div className="grid grid-cols-1 gap-x-4 gap-y-2 px-3 py-2 md:grid-cols-2">
            {Array.from({ length: REQUIRED_FIELDS }, (_, index) => (
              <div key={index} className="flex flex-col gap-1">
                <Skeleton className="h-2.5 w-20" />
                <Skeleton className="h-7 w-full" />
              </div>
            ))}
          </div>
        </div>
        <div className="border-b">
          <ReviewLabelSkeleton width="w-24" />
          <div className="px-3 py-2">
            {FIELD_SECTIONS.map((section, index) => (
              <div key={index} className="mb-1">
                <div className="flex items-center gap-2 px-2 py-1.5 text-xs">
                  <Skeleton className="size-3 rounded-sm" />
                  <Skeleton className={cn("inline-block h-2.5 align-middle", section.label)} />
                </div>
                <div className="ml-1">
                  {section.fields.map((width, row) => (
                    <div key={row} className="flex items-start gap-2 rounded-md px-2 py-1">
                      <Skeleton className="mt-1.5 size-1.5 shrink-0 rounded-full" />
                      <div className="flex min-w-0 flex-1 items-baseline gap-2">
                        <span className="text-2xs">
                          <Skeleton className="inline-block h-2 w-14 align-middle" />
                        </span>
                        <span className="text-xs">
                          <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="bg-muted/30 flex shrink-0 items-center justify-between border-t px-4 py-2.5">
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-28 align-middle" />
        </div>
        <Skeleton className="h-7 w-28" />
      </div>
    </div>
  );
}

export function ReconciliationWorkspaceSkeleton() {
  const t = useT();

  return (
    <div className="flex min-h-0 flex-1" aria-busy aria-label={t("Loading the reconciliation")}>
      <div className="contents" aria-hidden>
        <div className="min-w-0 flex-35">
          <DocumentPreviewPanelSkeleton />
        </div>
        <span className="bg-border w-px shrink-0 self-stretch" />
        <div className="min-w-0 flex-40">
          <ReviewColumnSkeleton />
        </div>
        <span className="bg-border w-px shrink-0 self-stretch" />
        <div className="min-w-0 flex-25">
          <AIActivityPanelSkeleton />
        </div>
      </div>
    </div>
  );
}
