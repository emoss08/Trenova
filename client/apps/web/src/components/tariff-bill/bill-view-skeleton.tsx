import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const LINE_WIDTHS = [46, 62, 38] as const;

/** `BillView`'s outline: its charge lines with their quantity × rate, and the totals. */
export function BillViewSkeleton({
  lines = LINE_WIDTHS.length,
  totalOnly = false,
  className,
}: {
  lines?: number;
  totalOnly?: boolean;
  className?: string;
}) {
  const t = useT();
  return (
    <div
      aria-busy
      aria-label={t("Loading the bill")}
      className={cn("flex flex-col gap-3", className)}
    >
      <ul aria-hidden className="divide-border flex flex-col divide-y">
        {Array.from({ length: lines }, (_, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          <li key={index} className="flex items-start justify-between gap-3 py-1.5">
            <span className="flex flex-1 flex-col gap-1">
              <Skeleton
                className="h-3.5"
                style={{ width: `${LINE_WIDTHS[index % LINE_WIDTHS.length]}%` }}
              />
              <Skeleton className="h-3 w-28" />
            </span>
            <Skeleton className="h-3.5 w-16" />
          </li>
        ))}
      </ul>
      <div aria-hidden className="flex flex-col gap-2 border-t pt-2">
        {!totalOnly && (
          <>
            <Skeleton className="h-3.5 w-full" />
            <Skeleton className="h-3.5 w-full" />
          </>
        )}
        <Skeleton className="h-4 w-full" />
      </div>
    </div>
  );
}
