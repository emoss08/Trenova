import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * Carrier invoice matching's loading shapes: the whole workspace while its code is read, and
 * the list's cards and the open invoice while the invoices and matches are. The workspace
 * opens on the invoices that need attention with the first one open, so that is what is drawn.
 */

const CARD_COUNT = 5;
const NUMBER_WIDTHS = ["w-24", "w-20", "w-28", "w-24", "w-16"] as const;
const BILL_TO_WIDTHS = ["w-40", "w-32", "w-44", "w-28", "w-36"] as const;
const FACT_WIDTHS = ["w-16", "w-20", "w-20", "w-20", "w-16", "w-24", "w-20", "w-32"] as const;

/**
 * The list's cards, invoices or matches alike: the number and its status, the line under it,
 * and the carrier and the amount along the bottom.
 */
export function MatchingListSkeleton({ label }: { label: string }) {
  return (
    <div className="flex flex-col gap-1.5 p-2" aria-busy aria-label={label}>
      {Array.from({ length: CARD_COUNT }, (_, index) => (
        <div key={index} className="rounded-lg border p-2.5 text-xs" aria-hidden>
          <div className="flex items-center justify-between gap-2">
            <Skeleton className={cn("inline-block h-2.5 align-middle", NUMBER_WIDTHS[index])} />
            <Skeleton className="h-5 w-16 rounded-full" />
          </div>
          <div className="mt-1">
            <Skeleton className={cn("inline-block h-2.5 align-middle", BILL_TO_WIDTHS[index])} />
          </div>
          <div className="mt-1 flex items-center justify-between">
            <Skeleton className="inline-block h-2.5 w-24 align-middle" />
            <Skeleton className="inline-block h-2.5 w-16 align-middle" />
          </div>
        </div>
      ))}
    </div>
  );
}

/** The open record: its facts in a titled panel, and a second panel with its actions. */
export function MatchingDetailSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4 p-4" aria-busy aria-label={t("Loading the details")}>
      <div className="contents" aria-hidden>
        <div className="bg-card flex flex-col overflow-hidden rounded-lg border">
          <div className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
            <Skeleton className="h-3 w-32" />
            <Skeleton className="h-2.5 w-48" />
          </div>
          <div className="grid grid-cols-2 gap-x-6 gap-y-3 p-3">
            {FACT_WIDTHS.map((width, index) => (
              <div key={index} className="flex flex-col">
                <div className="py-px text-xs">
                  <Skeleton className="inline-block h-2.5 w-20 align-middle" />
                </div>
                <div className="text-sm">
                  <Skeleton className={cn("inline-block h-3 align-middle", width)} />
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-card flex flex-col overflow-hidden rounded-lg border">
          <div className="flex min-h-9 items-center border-b px-3 py-1.5">
            <Skeleton className="h-3 w-24" />
          </div>
          <div className="p-3">
            <div className="mb-2 text-xs">
              <Skeleton className="inline-block h-2.5 w-full max-w-md align-middle" />
            </div>
            <div className="flex flex-wrap gap-2">
              <Skeleton className="h-7 w-28" />
              <Skeleton className="h-7 w-24" />
              <Skeleton className="h-7 w-28" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/** The whole workspace: the automation line, the figures, the list and the open invoice. */
export function MatchingWorkspaceSkeleton() {
  const t = useT();

  return (
    <div
      className="flex min-h-0 flex-1 flex-col gap-3"
      aria-busy
      aria-label={t("Loading carrier invoice matching")}
    >
      <div className="contents" aria-hidden>
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-96 max-w-full align-middle" />
        </div>
        <KpiStripSkeleton count={4} sub={false} />
        <div className="border-border bg-card grid min-h-120 flex-1 gap-0 overflow-hidden rounded-lg border md:grid-cols-[340px_1fr]">
          <div className="flex h-full min-h-0 flex-col overflow-hidden border-r">
            <div className="flex flex-col gap-2 border-b p-2">
              <div className="flex gap-1">
                <Skeleton className="h-6.5 flex-1" />
                <Skeleton className="h-6.5 flex-1" />
              </div>
              <Skeleton className="h-8 w-full" />
              <div className="flex flex-wrap gap-1">
                <Skeleton className="h-5 w-28 rounded-full" />
                <Skeleton className="h-5 w-10 rounded-full" />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-hidden">
              <MatchingListSkeleton label={t("Loading carrier invoices")} />
            </div>
          </div>
          <div className="min-h-0 overflow-hidden">
            <MatchingDetailSkeleton />
          </div>
        </div>
      </div>
    </div>
  );
}
