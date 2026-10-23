import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * `TariffInputsFields` while the tariff's declared inputs load: rows of a label beside its
 * control, and one full-width table, the shape a weight-break input takes.
 */
export function TariffInputsFieldsSkeleton({ fields = 4 }: { fields?: number }) {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the tariff's inputs…")}
      className="divide-border flex flex-col divide-y"
    >
      {Array.from({ length: fields }, (_, index) => (
        <div
          // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
          key={index}
          aria-hidden="true"
          className="grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] items-start gap-x-4 py-3 first:pt-0"
        >
          <div className="flex flex-col gap-1.5 pt-2">
            <Skeleton className="h-3 w-28" />
            <Skeleton className="h-2.5 w-40" />
          </div>
          <Skeleton className="h-7 w-full" />
        </div>
      ))}
      <div aria-hidden="true" className="flex flex-col gap-2 py-3 last:pb-0">
        <Skeleton className="h-3 w-32" />
        <Skeleton className="h-20 w-full" />
      </div>
    </div>
  );
}
