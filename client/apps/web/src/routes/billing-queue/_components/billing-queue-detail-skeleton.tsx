import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The detail pane while an item's code or record is read, measured from the pane itself
 * so nothing moves when the record lands: the header (the PRO row at the 2xl line, the
 * amount row at the 3xl line, "Open shipment" and the margin on the right), the Review
 * card's five checks and Nova's note, the tabs, the Charges tab's header, rows and
 * totals, and the action footer. Each text line sits in a box of that text's own line
 * height. The page draws it while the pane's code arrives and the pane draws it while
 * the item loads, so the two phases are one picture.
 */

const CHECK_WIDTHS = ["w-40", "w-52", "w-36", "w-44", "w-48"] as const;
const TAB_WIDTHS = ["w-16", "w-16", "w-20", "w-14", "w-14"] as const;
const CHARGE_ROWS = [
  { name: "w-24", basis: "w-40", amount: "w-16" },
  { name: "w-28", basis: "w-36", amount: "w-14" },
  { name: "w-20", basis: "w-44", amount: "w-14" },
] as const;
const TOTAL_ROWS = ["w-14", "w-20", "w-10", "w-20", "w-12"] as const;

/** A line of text as a box of its line height holding a bar of its glyphs' height. */
function Line({ height, bar, width }: { height: string; bar: string; width: string }) {
  return (
    <span className={cn("flex items-center", height)}>
      <Skeleton className={cn(bar, width)} />
    </span>
  );
}

export function BillingQueueDetailSkeleton() {
  const t = useT();

  return (
    <div
      className="flex h-full flex-col"
      aria-busy
      aria-label={t("Loading the billing queue item")}
    >
      <div className="contents" aria-hidden>
        <div className="flex shrink-0 flex-col gap-3 px-5 pt-4 pb-3">
          <div className="flex h-6.5 items-center gap-2.5">
            <Skeleton className="h-5 w-36" />
            <Skeleton className="rounded-badge h-5 w-18" />
            <Skeleton className="h-3 w-44" />
            <span className="flex-1" />
            <Skeleton className="h-3 w-24" />
          </div>
          <div className="flex h-7.5 items-center gap-3.5">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-3 w-48" />
            <span className="flex-1" />
            <Skeleton className="h-3 w-28" />
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-5.5 overflow-hidden px-5 pt-1 pb-6">
          <div className="border-border rounded-md border">
            <div className="border-border flex h-9.5 items-center justify-between border-b px-3">
              <Skeleton className="h-3 w-14" />
              <Skeleton className="h-2.5 w-20" />
            </div>
            <div className="flex flex-col py-1">
              {CHECK_WIDTHS.map((width, index) => (
                <div key={index} className="mx-1 flex h-7.25 items-center gap-2.5 px-2">
                  <Skeleton className="size-4 rounded-full" />
                  <Skeleton className="h-3 w-16" />
                  <Skeleton className={cn("h-3", width)} />
                </div>
              ))}
            </div>
            <div className="border-border bg-muted flex gap-2.5 rounded-b-md border-t px-3 py-2.5">
              <Skeleton className="mt-0.5 size-3.5 rounded-full" />
              <div className="flex min-w-0 flex-1 flex-col gap-1.5">
                <div className="flex flex-col">
                  <Line height="h-4.5" bar="h-3" width="w-full" />
                  <Line height="h-4.5" bar="h-3" width="w-3/5" />
                </div>
                <Line height="h-4" bar="h-2.5" width="w-40" />
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-4">
            <div className="border-border flex h-8.25 items-center gap-4.5 border-b">
              {TAB_WIDTHS.map((width, index) => (
                <Skeleton key={index} className={cn("h-3", width)} />
              ))}
            </div>
            <div className="flex flex-col gap-2">
              <div className="flex h-6 items-center gap-2">
                <Skeleton className="h-3 w-14" />
                <span className="flex-1" />
                <Skeleton className="h-2.5 w-36" />
                <Skeleton className="h-6 w-16" />
              </div>
              <div className="border-border rounded-md border">
                <div className="border-border flex h-8 items-center border-b px-2.5">
                  <Skeleton className="h-2.5 w-12" />
                </div>
                {CHARGE_ROWS.map((row, index) => (
                  <div
                    key={index}
                    className="border-border flex h-10.75 items-center justify-between gap-2 border-b px-2.5 py-1"
                  >
                    <div className="flex flex-col gap-px">
                      <Line height="h-4.25" bar="h-3" width={row.name} />
                      <Line height="h-4" bar="h-2.5" width={row.basis} />
                    </div>
                    <Skeleton className={cn("h-3", row.amount)} />
                  </div>
                ))}
                <div className="flex items-start justify-between gap-4 p-2.5">
                  <Skeleton className="h-6 w-24" />
                  <div className="flex min-w-60 flex-col gap-1">
                    {TOTAL_ROWS.map((width, index) => (
                      <div
                        key={index}
                        className={cn(
                          "flex h-4.25 items-center justify-between gap-4",
                          index === 2 && "border-border box-content border-t pt-1.5",
                        )}
                      >
                        <Skeleton className={cn("h-3", width)} />
                        <Skeleton className="h-3 w-20" />
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="border-border flex h-12.25 shrink-0 items-center justify-end gap-2 border-t px-5">
          <Skeleton className="h-7 w-24" />
          <Skeleton className="h-7 w-28" />
          <Skeleton className="h-7 w-16" />
          <Skeleton className="h-7 w-20" />
        </div>
      </div>
    </div>
  );
}

/** The pane with nothing chosen: the sketch of a record and how to open one. */
export function BillingQueueDetailUnselected() {
  const t = useT();

  return (
    <BillingDetailUnselected
      layout="tabs"
      title={t("Nothing to open")}
      description={t("No item in the queue matches what you're looking at.")}
    />
  );
}
