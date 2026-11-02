import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

export const STOP_GRID =
  "grid grid-cols-[1.75rem_minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)_6rem] items-center gap-2.5";

const STOP_ROWS = [
  ["w-28", "w-36", "w-24"],
  ["w-32", "w-28", "w-24"],
  ["w-24", "w-40", "w-24"],
] as const;

const FACT_WIDTHS = [
  "w-20",
  "w-16",
  "w-20",
  "w-24",
  "w-20",
  "w-24",
  "w-16",
  "w-20",
  "w-16",
  "w-24",
  "w-28",
  "w-20",
] as const;

export function BillingQueueShipmentTabSkeleton() {
  const t = useT();

  return (
    <div className="flex flex-col gap-5.5" aria-busy aria-label={t("Loading the shipment")}>
      <div className="contents" aria-hidden>
        <section className="flex flex-col gap-2">
          <div className="flex items-center gap-2">
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-2.5 w-40" />
          </div>
          <div className="rounded-md border border-border">
            <div className={`${STOP_GRID} h-8 border-b border-border px-3`}>
              <span />
              <Skeleton className="h-2.5 w-10" />
              <Skeleton className="h-2.5 w-16" />
              <Skeleton className="h-2.5 w-24" />
              <Skeleton className="ml-auto h-2.5 w-12" />
            </div>
            {STOP_ROWS.map(([name, place, time], index) => (
              <div
                key={index}
                className={`${STOP_GRID} border-border px-3 py-2.5 ${index < STOP_ROWS.length - 1 ? "border-b" : ""}`}
              >
                <Skeleton className="size-5 rounded-full" />
                <span className="flex flex-col gap-1.5">
                  <Skeleton className={`h-3 ${name}`} />
                  <Skeleton className={`h-2.5 ${place}`} />
                </span>
                <Skeleton className={`h-3 ${time}`} />
                <Skeleton className="h-3 w-28" />
                <Skeleton className="ml-auto h-3 w-12" />
              </div>
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <Skeleton className="h-3 w-10" />
          <div className="grid grid-cols-2 gap-x-6 gap-y-3 rounded-md border border-border px-3.5 py-3 md:grid-cols-3">
            {FACT_WIDTHS.map((width, index) => (
              <span key={index} className="flex flex-col gap-1.5">
                <Skeleton className="h-2.5 w-14" />
                <Skeleton className={`h-3 ${width}`} />
              </span>
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-2">
          <Skeleton className="h-3 w-28" />
          <div className="flex items-center gap-2.5 rounded-md border border-border px-3.5 py-2.5">
            <span className="flex flex-1 flex-col gap-1.5">
              <Skeleton className="h-3 w-72" />
              <Skeleton className="h-2.5 w-96 max-w-full" />
            </span>
            <Skeleton className="h-6.5 w-24" />
          </div>
        </section>
      </div>
    </div>
  );
}
