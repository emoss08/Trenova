import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";

/**
 * The day's brief before it is read: the greeting line at the brief's own type size and the
 * "as of" line under it. A workspace without an AI provider has no brief, so it draws nothing.
 */
export function ShipmentBriefingSkeleton() {
  const t = useT();
  const { ai } = useShipmentCapabilities();

  if (!ai) return null;

  return (
    <div className="flex min-w-0 flex-col gap-1" aria-busy aria-label={t("Loading the brief")}>
      <div className="contents" aria-hidden>
        <div className="text-lg leading-relaxed">
          <Skeleton className="inline-block h-4 w-2/3 align-middle" />
        </div>
        <div className="text-xs">
          <Skeleton className="inline-block h-2.5 w-24 align-middle" />
        </div>
      </div>
    </div>
  );
}
