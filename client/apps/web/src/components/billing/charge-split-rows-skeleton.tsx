import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** The split rows, their bar and their status line while the rules module that measures them loads. */
export function ChargeSplitRowsSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Measuring the split…")} className="flex flex-col gap-2">
      <div className="flex h-7 items-center justify-between gap-2">
        <Skeleton aria-hidden="true" className="h-6 w-36" />
        <Skeleton aria-hidden="true" className="h-5 w-20" />
      </div>
      <Skeleton aria-hidden="true" className="h-1.5 w-full rounded-full" />
      {[0, 1].map((row) => (
        <div
          key={row}
          className="grid grid-cols-[0.5rem_minmax(0,1fr)_7.5rem_5.5rem_1.75rem] items-center gap-2"
        >
          <Skeleton aria-hidden="true" className="size-2 rounded-full" />
          <Skeleton aria-hidden="true" className="h-7 w-full" />
          <Skeleton aria-hidden="true" className="h-7 w-full" />
          <Skeleton aria-hidden="true" className="h-3 w-full" />
          <Skeleton aria-hidden="true" className="size-7" />
        </div>
      ))}
      <Skeleton aria-hidden="true" className="h-3 w-44" />
    </div>
  );
}
