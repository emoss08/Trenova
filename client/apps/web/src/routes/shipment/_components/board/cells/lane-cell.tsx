import { useT } from "@trenova/shared/i18n/use-t";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { getDestinationLocation, getOriginLocation, getRouteProgress } from "@/lib/shipment-utils";
import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useShipmentBoardStore } from "../store";

export function LaneCell({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const highlighted = useShipmentBoardStore((state) => state.highlightId === shipment.id);
  const origin = getOriginLocation(shipment);
  const destination = getDestinationLocation(shipment);
  const progress = getRouteProgress(shipment, Date.now() / 1000);
  const stage = shipment.stage ?? "NeedsCoverage";
  const moving = stage === "Moving" || stage === "Late";

  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <div className="flex min-w-0 items-center gap-1.5">
        <span className={cn("truncate text-sm font-medium", highlighted && "text-brand")}>
          {origin?.city ?? origin?.code ?? "—"}
        </span>
        <ArrowRightIcon className="text-muted-foreground size-3 shrink-0" aria-hidden />
        <span className="truncate text-sm font-medium">
          {destination?.city ?? destination?.code ?? "—"}
        </span>
        <span
          className="bg-muted ml-1 h-1 w-10 shrink-0 overflow-hidden rounded-full"
          role="progressbar"
          aria-label={t("Route progress")}
          aria-valuenow={progress.percent}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <span
            className={cn("block h-full rounded-full", STAGE_META[stage].swatchClassName)}
            style={{ width: `${progress.percent}%` }}
          />
        </span>
      </div>
      <span className="text-muted-foreground font-mono text-xs tabular-nums in-data-[density=compact]:hidden">
        {moving
          ? t("{0} mi left", progress.milesLeft.toLocaleString())
          : t("{0} mi", progress.totalMiles.toLocaleString())}
      </span>
    </div>
  );
}
