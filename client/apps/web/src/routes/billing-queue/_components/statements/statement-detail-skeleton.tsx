import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * A customer's statement while its code or its record is read: the customer and the period over
 * the live total, the three steps of the cycle, the schedule line and the invoice cards it
 * becomes (the first opened onto its shipments), and the bar that bills it. The page draws it
 * while the pane's code arrives and the pane draws it while the statement loads.
 */

const STEP_WIDTHS = [
  { label: "w-16", detail: "w-28" },
  { label: "w-12", detail: "w-20" },
  { label: "w-20", detail: "w-24" },
] as const;
const GROUPS = [
  { label: "w-32", shipments: 3 },
  { label: "w-24", shipments: 0 },
] as const;
const SHIPMENT_WIDTHS = ["w-20", "w-24", "w-16"] as const;

export function StatementDetailSkeleton() {
  const t = useT();

  return (
    <div
      className="bg-card flex h-full min-h-0 flex-col overflow-hidden"
      aria-busy
      aria-label={t("Loading the statement")}
    >
      <div className="contents" aria-hidden>
        <div className="flex items-center gap-2 border-b px-3 py-2">
          <div className="min-w-0">
            <div className="text-sm">
              <Skeleton className="inline-block h-3 w-40 align-middle" />
            </div>
            <div className="text-xs">
              <Skeleton className="inline-block h-2.5 w-32 align-middle" />
            </div>
          </div>
          <Skeleton className="ml-auto h-4 w-20" />
        </div>

        <ol className="flex items-stretch gap-1 border-b px-3 py-2">
          {STEP_WIDTHS.map((step, index) => (
            <li key={index} className="flex min-w-0 flex-1 items-center gap-2">
              <Skeleton className="size-5 shrink-0 rounded-full" />
              <span className="min-w-0">
                <span className="block text-xs">
                  <Skeleton className={cn("inline-block h-2.5 align-middle", step.label)} />
                </span>
                <span className="block text-xs">
                  <Skeleton className={cn("inline-block h-2.5 align-middle", step.detail)} />
                </span>
              </span>
              {index < STEP_WIDTHS.length - 1 ? (
                <span className="bg-border ml-auto hidden h-px flex-1 sm:block" />
              ) : null}
            </li>
          ))}
        </ol>

        <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden p-2">
          <div className="px-1 text-xs">
            <Skeleton className="inline-block h-2.5 w-64 max-w-full align-middle" />
          </div>
          {GROUPS.map((group, index) => (
            <div key={index} className="bg-card overflow-hidden rounded-lg border">
              <div className="flex items-center gap-2 px-3 py-2">
                <Skeleton className="size-4 shrink-0 rounded-sm" />
                <span className="flex min-w-0 flex-1 items-center gap-2 text-sm">
                  <Skeleton className="size-3.5 shrink-0 rounded-sm" />
                  <Skeleton className="size-3.5 shrink-0 rounded-sm" />
                  <Skeleton className={cn("inline-block h-3 align-middle", group.label)} />
                </span>
                <Skeleton className="h-2.5 w-16 shrink-0" />
                <Skeleton className="h-3 w-16 shrink-0" />
              </div>
              {group.shipments > 0 ? (
                <div className="divide-y border-t">
                  {SHIPMENT_WIDTHS.slice(0, group.shipments).map((width, row) => (
                    <div key={row} className="flex items-center gap-2 px-3 py-1.5 text-xs">
                      <Skeleton className="size-4 shrink-0 rounded-sm" />
                      <span className="w-28 shrink-0">
                        <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
                      </span>
                      <span className="hidden w-28 shrink-0 sm:inline">
                        <Skeleton className="inline-block h-2.5 w-20 align-middle" />
                      </span>
                      <Skeleton className="ml-auto h-2.5 w-14 shrink-0" />
                    </div>
                  ))}
                </div>
              ) : null}
            </div>
          ))}
        </div>

        <div className="bg-muted/40 flex items-center gap-2 border-t p-2">
          <Skeleton className="h-2.5 w-24" />
          <Skeleton className="ml-auto h-7 w-20" />
        </div>
      </div>
    </div>
  );
}

/** The pane with no customer chosen: the sketch of a statement and what opening one shows. */
export function StatementDetailUnselected() {
  const t = useT();

  return (
    <BillingDetailUnselected
      layout="cards"
      title={t("Pick a statement")}
      description={t(
        "See what a customer has accumulated this period, which invoices it becomes, and bill it early if you have to.",
      )}
    />
  );
}
