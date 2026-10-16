import { BillingDetailUnselected } from "@/components/billing/billing-empty";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The detail pane while an item's code or record is read: the PRO number and its status, the
 * amount, the row of facts, the action bar, the four tabs and the charges tab it opens on. Each
 * text line sits in a box of that text's own line height, so the record lands where its outline
 * was. The page draws it while the pane's code arrives and the pane draws it while the item
 * loads, so the two phases are one picture.
 */

const FACT_WIDTHS = ["w-16", "w-28", "w-20", "w-36"] as const;
const ACTION_WIDTHS = ["w-24", "w-28", "w-20"] as const;
const TAB_WIDTHS = ["w-14", "w-18", "w-18", "w-14"] as const;
const ACCESSORIAL_ROWS = [
  { name: "w-28", detail: "w-20", amount: "w-14" },
  { name: "w-20", detail: "w-24", amount: "w-12" },
] as const;

export function BillingQueueDetailSkeleton() {
  const t = useT();

  return (
    <div
      className="flex h-full flex-col"
      aria-busy
      aria-label={t("Loading the billing queue item")}
    >
      <div className="contents" aria-hidden>
        <div className="shrink-0 space-y-3 border-b px-4 py-4">
          <div className="flex items-center gap-2 text-lg">
            <Skeleton className="inline-block h-4 w-28 align-middle" />
            <Skeleton className="h-5 w-20 rounded-badge" />
          </div>
          <div className="flex h-5 items-center">
            <Skeleton className="h-5 w-32" />
          </div>
          <div className="grid grid-cols-2 gap-x-6 gap-y-2 sm:grid-cols-3 lg:grid-cols-4">
            {FACT_WIDTHS.map((width, index) => (
              <div key={index}>
                <div className="text-xs">
                  <Skeleton className="inline-block h-2.5 w-12 align-middle" />
                </div>
                <div className="text-sm">
                  <Skeleton className={cn("inline-block h-3 align-middle", width)} />
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="flex items-center gap-2 border-b px-4 py-2">
          {ACTION_WIDTHS.map((width, index) => (
            <Skeleton key={index} className={cn("h-7", width)} />
          ))}
        </div>
        <div className="flex min-h-0 flex-1 flex-col gap-2">
          <div className="flex w-full items-center gap-x-0.5 border-b py-1">
            {TAB_WIDTHS.map((width, index) => (
              <span key={index} className="flex h-8 grow items-center justify-center px-2.5">
                <Skeleton className={cn("h-3", width)} />
              </span>
            ))}
          </div>
          <ChargesSkeleton />
        </div>
      </div>
    </div>
  );
}

/** The charges tab: the rating line, the base rate, the accessorials and the total. */
function ChargesSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-4">
      <div className="flex items-center justify-between rounded-md border px-3 py-2">
        <Skeleton className="h-3 w-48" />
        <Skeleton className="h-6 w-20" />
      </div>
      <div className="flex flex-col">
        <ChargeRowSkeleton name="w-20" detail="w-32" amount="w-16" />
        <div className="flex items-center justify-between p-2">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-6 w-16" />
        </div>
        {ACCESSORIAL_ROWS.map((row, index) => (
          <ChargeRowSkeleton key={index} {...row} />
        ))}
      </div>
      <div className="bg-muted/50 flex items-center justify-between rounded-md px-3 py-2.5">
        <Skeleton className="h-3.5 w-10" />
        <Skeleton className="h-4 w-20" />
      </div>
    </div>
  );
}

function ChargeRowSkeleton({
  name,
  detail,
  amount,
}: {
  name: string;
  detail: string;
  amount: string;
}) {
  return (
    <div className="flex items-center justify-between gap-2 p-2">
      <div className="flex flex-col">
        <div className="text-sm">
          <Skeleton className={cn("inline-block h-3 align-middle", name)} />
        </div>
        <div className="text-xs">
          <Skeleton className={cn("inline-block h-2.5 align-middle", detail)} />
        </div>
      </div>
      <Skeleton className={cn("h-3 shrink-0", amount)} />
    </div>
  );
}

/** The pane with nothing chosen: the sketch of a record and how to open one. */
export function BillingQueueDetailUnselected() {
  const t = useT();

  return (
    <BillingDetailUnselected
      layout="tabs"
      title={t("Nothing open")}
      description={t(
        "Pick an item from the queue to review it here, or press J to start at the top.",
      )}
    />
  );
}
