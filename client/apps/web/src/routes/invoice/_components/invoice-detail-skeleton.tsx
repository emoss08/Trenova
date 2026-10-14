import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * An invoice while its code or its record is read: the number with its badges and actions, the
 * amount beside the customer, the four facts, the route strip, the seven tabs and the overview
 * they open on. The page draws it while the pane's code arrives and the pane draws it while the
 * invoice loads, so the two phases are one picture.
 */

const BADGE_WIDTHS = ["w-16", "w-14", "w-20"] as const;
const FACT_WIDTHS = ["w-20", "w-20", "w-14", "w-16"] as const;
const TAB_WIDTHS = ["w-16", "w-14", "w-14", "w-16", "w-14", "w-18", "w-14"] as const;
const TOTAL_ROWS = ["w-16", "w-20", "w-12"] as const;

export function InvoiceDetailSkeleton() {
  const t = useT();

  return (
    <div className="flex h-full flex-col" aria-busy aria-label={t("Loading the invoice")}>
      <div className="contents" aria-hidden>
        <div className="shrink-0 space-y-4 border-b px-4 py-4">
          <div className="flex flex-wrap items-center justify-between">
            <div className="flex items-center gap-2 text-lg">
              <Skeleton className="inline-block h-4 w-28 align-middle" />
              {BADGE_WIDTHS.map((width, index) => (
                <Skeleton key={index} className={cn("h-5 rounded-full", width)} />
              ))}
            </div>
            <div className="flex items-center gap-2">
              <Skeleton className="h-7 w-24" />
              <Skeleton className="h-7 w-16" />
              <Skeleton className="h-7 w-20" />
              <Skeleton className="size-7" />
            </div>
          </div>
          <div className="flex items-baseline gap-3">
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-3 w-36" />
          </div>
          <div className="grid grid-cols-2 gap-x-6 gap-y-3 md:grid-cols-4">
            {FACT_WIDTHS.map((width, index) => (
              <div key={index} className="flex flex-col">
                <div className="py-px text-xs">
                  <Skeleton className="inline-block h-2.5 w-16 align-middle" />
                </div>
                <div className="text-sm">
                  <Skeleton className={cn("inline-block h-3 align-middle", width)} />
                </div>
              </div>
            ))}
          </div>
        </div>
        <Skeleton className="h-32 w-full shrink-0 rounded-none border-b" />
        <div className="flex min-h-0 flex-1 flex-col gap-2">
          <div className="flex w-full items-center gap-x-0.5 border-b py-1">
            {TAB_WIDTHS.map((width, index) => (
              <span key={index} className="flex h-8 grow items-center justify-center px-2.5">
                <Skeleton className={cn("h-3", width)} />
              </span>
            ))}
          </div>
          <div className="grid gap-5 px-4 py-2 xl:grid-cols-2">
            <div className="flex flex-col gap-5">
              <div className="bg-card rounded-lg border p-3">
                <Skeleton className="h-2.5 w-14" />
                <div className="mt-1.5 flex flex-col gap-1.5">
                  <Skeleton className="h-3 w-40" />
                  <Skeleton className="h-2.5 w-48" />
                  <Skeleton className="h-2.5 w-32" />
                </div>
              </div>
              <div className="bg-card rounded-lg border p-3">
                <Skeleton className="h-2.5 w-16" />
                <div className="mt-1 flex flex-col">
                  {TOTAL_ROWS.map((width, index) => (
                    <div key={index} className="flex items-center justify-between gap-4 py-1.5">
                      <Skeleton className={cn("h-2.5", width)} />
                      <Skeleton className="h-3 w-16" />
                    </div>
                  ))}
                </div>
              </div>
            </div>
            <div className="bg-card rounded-lg border p-3">
              <Skeleton className="h-2.5 w-20" />
              <div className="mt-2 grid grid-cols-2 gap-x-6 gap-y-3">
                {FACT_WIDTHS.map((width, index) => (
                  <div key={index} className="flex flex-col gap-1">
                    <Skeleton className="h-2.5 w-14" />
                    <Skeleton className={cn("h-3", width)} />
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

/** The pane with no invoice chosen: the sketch of one and what opening it shows. */
export function InvoiceDetailUnselected() {
  const t = useT();

  return (
    <BillingDetailUnselected
      layout="tabs"
      title={t("Nothing open")}
      description={t(
        "Pick an invoice from the list to review its charges, documents and what has been sent or paid.",
      )}
    />
  );
}
