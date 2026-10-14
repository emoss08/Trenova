import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/**
 * The transfer dialog's loading shapes. The dialog is mounted only once somebody opens it, so
 * while its code arrives it is already meant to be on screen: the frame opens at once with its
 * title, and the body takes the shape of what it will show — the shipment picker, or, when the
 * address names a run to reattach to, that run's progress.
 */

const PICKER_ROWS = 6;
const PRO_WIDTHS = ["w-20", "w-24", "w-16", "w-24", "w-20", "w-16"] as const;
const CUSTOMER_WIDTHS = ["w-32", "w-24", "w-28", "w-20", "w-36", "w-24"] as const;
const CANDIDATE_GRID =
  "grid grid-cols-[1.5rem_minmax(0,1.2fr)_minmax(0,1.4fr)_7.5rem_6rem_6rem] items-center gap-3";

/** The picker's rows while the shipments that can transfer are read. */
export function BulkBillingTransferCandidateRowsSkeleton() {
  const t = useT();

  return (
    <ul className="divide-y" aria-busy aria-label={t("Loading shipments")}>
      {Array.from({ length: PICKER_ROWS }, (_, index) => (
        <li key={index} className={cn(CANDIDATE_GRID, "px-3 py-2 text-xs")} aria-hidden>
          <Skeleton className="size-4 rounded-sm" />
          <span className="flex min-w-0 flex-col">
            <span>
              <Skeleton className={cn("inline-block h-2.5 align-middle", PRO_WIDTHS[index])} />
            </span>
            <span>
              <Skeleton className="inline-block h-2.5 w-14 align-middle" />
            </span>
          </span>
          <Skeleton className={cn("h-2.5", CUSTOMER_WIDTHS[index])} />
          <Skeleton className="h-5 w-20 rounded-full" />
          <Skeleton className="h-2.5 w-16" />
          <Skeleton className="ml-auto h-2.5 w-14" />
        </li>
      ))}
    </ul>
  );
}

/** The picker: the search and status filter, the count, and the table with its rows loading. */
function PickerSkeleton() {
  return (
    <div className="flex min-h-0 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2" aria-hidden>
        <Skeleton className="h-8 min-w-48 flex-1" />
        <Skeleton className="h-8 w-72 rounded-lg" />
      </div>
      <div className="flex items-center text-xs" aria-hidden>
        <Skeleton className="inline-block h-2.5 w-36 align-middle" />
      </div>
      <div className="overflow-hidden rounded-lg border">
        <div className={cn(CANDIDATE_GRID, "bg-muted/40 border-b px-3 py-2 text-xs")} aria-hidden>
          <Skeleton className="size-4 rounded-sm" />
          <Skeleton className="h-2.5 w-8" />
          <Skeleton className="h-2.5 w-16" />
          <Skeleton className="h-2.5 w-12" />
          <Skeleton className="h-2.5 w-14" />
          <Skeleton className="ml-auto h-2.5 w-10" />
        </div>
        <div className="h-[min(26rem,50vh)] overflow-hidden">
          <BulkBillingTransferCandidateRowsSkeleton />
        </div>
      </div>
    </div>
  );
}

/** A run while it is read back: the progress line, its bar, the two counts and the note. */
export function BulkBillingTransferRunSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-4 py-10" aria-busy aria-label={t("Loading the transfer")}>
      <div className="contents" aria-hidden>
        <div className="flex flex-col gap-2">
          <div className="flex items-baseline justify-between gap-4">
            <div className="text-sm">
              <Skeleton className="inline-block h-3 w-64 align-middle" />
            </div>
            <div className="text-xs">
              <Skeleton className="inline-block h-2.5 w-32 align-middle" />
            </div>
          </div>
          <Skeleton className="h-2 w-full rounded-full" />
        </div>
        <div className="grid grid-cols-2 gap-3 text-xs">
          {[0, 1].map((index) => (
            <div key={index} className="rounded-lg border px-3 py-2">
              <div>
                <Skeleton className="inline-block h-2.5 w-20 align-middle" />
              </div>
              <div className="text-lg">
                <Skeleton className="inline-block h-4 w-10 align-middle" />
              </div>
            </div>
          ))}
        </div>
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-96 max-w-full align-middle" />
        </div>
      </div>
    </div>
  );
}

export function BulkBillingTransferDialogSkeleton({
  open,
  onOpenChange,
  runId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  runId: string | null;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="flex max-h-[90vh] flex-col">
        <DialogHeader>
          <DialogTitle>{t("Transfer to billing")}</DialogTitle>
          <DialogDescription>
            {t(
              "Each shipment runs the billing readiness check before it moves into the queue. Completed shipments that pass are marked Ready to Invoice first.",
            )}
          </DialogDescription>
        </DialogHeader>
        {runId ? <BulkBillingTransferRunSkeleton /> : <PickerSkeleton />}
        <DialogFooter className="flex-wrap items-center gap-2 sm:justify-between">
          {runId ? null : (
            <div className="contents" aria-hidden>
              <Skeleton className="h-2.5 w-16" />
              <div className="flex items-center gap-2">
                <Skeleton className="h-8 w-18" />
                <Skeleton className="h-8 w-36" />
              </div>
            </div>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
