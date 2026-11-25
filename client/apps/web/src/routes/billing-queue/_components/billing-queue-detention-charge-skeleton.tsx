import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = ["w-44", "w-36"] as const;

/**
 * The detention charges behind a shipment's detention line while they load, drawn as the
 * list lays out: each stop's facility and standing, its amount and its actions.
 */
export function BillingQueueDetentionChargeSkeleton() {
  const t = useT();

  return (
    <ul className="flex flex-col" aria-busy aria-label={t("Loading the detention charges")}>
      {ROWS.map((width) => (
        <li
          key={width}
          aria-hidden
          className="border-border flex items-center justify-between gap-3 border-b py-2.5 last:border-b-0"
        >
          <span className="flex flex-col gap-1.5">
            <Skeleton className={`h-3.5 ${width}`} />
            <Skeleton className="h-2.5 w-28" />
          </span>
          <span className="flex items-center gap-2">
            <Skeleton className="h-3.5 w-16" />
            <Skeleton className="h-6 w-16" />
            <Skeleton className="h-6 w-14" />
          </span>
        </li>
      ))}
    </ul>
  );
}
