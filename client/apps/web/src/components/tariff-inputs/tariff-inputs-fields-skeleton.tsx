import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * `TariffInputsFields` while the tariff's declared inputs load: a two-column grid of labelled
 * fields and one full-width table, the shape a weight-break input takes.
 */
export function TariffInputsFieldsSkeleton({ fields = 4 }: { fields?: number }) {
  const t = useT();
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the tariff's inputs…")}
      className="grid grid-cols-1 gap-4 md:grid-cols-2"
    >
      {Array.from({ length: fields }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <div key={index} aria-hidden="true" className="flex flex-col gap-1.5">
          <Skeleton className="h-3 w-28" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-2.5 w-40" />
        </div>
      ))}
      <div aria-hidden="true" className="flex flex-col gap-2 rounded-md border p-3 md:col-span-2">
        <Skeleton className="h-3.5 w-32" />
        <Skeleton className="h-20 w-full" />
      </div>
    </div>
  );
}
