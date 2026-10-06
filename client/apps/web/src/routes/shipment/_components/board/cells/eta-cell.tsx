import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan, formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { getDestinationStop } from "@/lib/shipment-utils";
import type { Shipment } from "@trenova/shared/types/shipment";

/** How far off its window a load will land, phrased for the ETA column and the open row. */
export function useEtaDelta(shipment: Shipment): { text: string; tone: string } | null {
  const t = useT();
  const slack = shipment.eta?.slackMinutes;
  if (shipment.stage === "Delivered" || slack == null) return null;
  if (slack < 0) return { text: `+${formatMinutesSpan(-slack)}`, tone: "text-danger" };
  if (slack < 15) return { text: t("on time"), tone: "text-success" };
  return { text: t("{0} early", formatMinutesSpan(slack)), tone: "text-success" };
}

export function EtaCell({ shipment }: { shipment: Shipment }) {
  const stop = getDestinationStop(shipment);
  const arrival =
    shipment.eta?.estimatedArrival ?? stop?.scheduledWindowEnd ?? stop?.scheduledWindowStart ?? null;
  const delta = useEtaDelta(shipment);

  if (!arrival) {
    return <span className="text-muted-foreground font-mono text-sm">—</span>;
  }

  return (
    <div className="flex min-w-0 flex-col">
      <span className="font-mono text-sm tabular-nums">
        {formatToUserTimezone(arrival, { showTimeZone: false, showSeconds: false })}
      </span>
      {delta ? (
        <span
          className={cn(
            "in-data-[density=compact]:hidden font-mono text-xs tabular-nums",
            delta.tone,
          )}
        >
          {delta.text}
        </span>
      ) : null}
    </div>
  );
}
