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
          <Skeleton className="h-5 w-20 rounded-badge" />
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

/**
 * A run while it is read back: the status line over the stage, with the delivered stack on
 * the left, the desk and its four steps in the middle, and the two outcome stacks on the right.
 */
export function BulkBillingTransferRunSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-3.5" aria-busy aria-label={t("Loading the transfer")}>
      <div className="contents" aria-hidden>
        <div className="flex items-baseline justify-between gap-4 text-xs">
          <Skeleton className="inline-block h-2.5 w-48 align-middle" />
          <Skeleton className="inline-block h-2.5 w-24 align-middle" />
        </div>
        <div className="relative h-75 overflow-hidden rounded-lg border [--side:min(40%,310px)] max-sm:[--side:38%]">
          <SkeletonPile className="top-[169px] left-[calc(50%-var(--side))]" />
          <div className="absolute top-1/2 left-1/2 flex translate-x-[-40px] -translate-y-1/2 items-center gap-6">
            <Skeleton className="h-26 w-20 rounded-[5px]" />
            <div className="flex w-32 flex-col gap-0.5 max-sm:w-auto">
              {STEP_WIDTHS.map((width) => (
                <div
                  key={width}
                  className="grid grid-cols-[12px_minmax(0,1fr)] items-center gap-2 px-1.5 py-1 text-xs"
                >
                  <Skeleton className="size-3 rounded-full" />
                  <span className="max-sm:hidden">
                    <Skeleton className={cn("inline-block h-2.5 align-middle", width)} />
                  </span>
                </div>
              ))}
            </div>
          </div>
          <SkeletonPile className="top-[92px] left-[calc(50%+var(--side))]" />
          <SkeletonPile className="top-[208px] left-[calc(50%+var(--side))]" />
        </div>
      </div>
    </div>
  );
}

const STEP_WIDTHS = ["w-10", "w-16", "w-8", "w-20"] as const;

function SkeletonPile({ className }: { className: string }) {
  return (
    <div
      className={cn(
        "absolute flex -translate-x-1/2 -translate-y-1/2 flex-col items-center gap-2.5 px-2.5 py-1.5",
        className,
      )}
    >
      <Skeleton className="h-[58px] w-11 rounded-[3px]" />
      <span className="text-2xs">
        <Skeleton className="inline-block h-2 w-20 align-middle" />
      </span>
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
