import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const LINES = [40, 56, 48] as const;

/**
 * The shipment's charge lines while the rules module that prices them loads: the line haul,
 * the accessorials header and a few charges, each a name over its amount.
 */
export function BillingQueueChargeLinesSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading the charges…")} className="flex flex-col gap-1">
      {LINES.map((width, index) => (
        <div key={index} aria-hidden="true" className="flex items-center justify-between p-2">
          <Skeleton className="h-3.5" style={{ width: `${width}%` }} />
          <Skeleton className="h-3.5 w-16" />
        </div>
      ))}
    </div>
  );
}
