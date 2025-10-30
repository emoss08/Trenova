import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopSchema } from "@/lib/schemas/stop-schema";
import { formatLocation } from "@/lib/utils";
export function ShipmentMovePreview({
  moves,
}: {
  moves: ShipmentSchema["moves"];
}) {
  return (
    <div className="flex flex-col gap-2">
      {moves.map((move) => (
        <div key={move.id} className="flex flex-col justify-between gap-2">
          <div className="flex items-center gap-2">
            Move {move.sequence + 1}
          </div>
          <div className="flex flex-col gap-2">
            {[...move.stops]
              .sort((a, b) => a.sequence - b.sequence)
              .map((stop) => (
                <ShipmentStopPreview key={stop.id} stop={stop} />
              ))}
          </div>
        </div>
      ))}
    </div>
  );
}

function ShipmentStopPreview({ stop }: { stop: StopSchema }) {
  return (
    <div className="flex flex-row items-center gap-1 h-6">
      <div className="border border-border rounded-md p-2 size-6 items-center shrink-0 flex justify-center">
        <span className="text-xs font-medium text-foreground">
          {stop.sequence + 1}
        </span>
      </div>
      <div className="flex flex-col leading-tight items-start">
        <div className="flex flex-row items-center gap-1">
          <p className="text-sm text-foreground truncate max-w-[300px]">
            {stop.location?.name}
          </p>
          <p className="text-2xs text-muted-foreground truncate max-w-[100px]">
            ({stop.type})
          </p>
        </div>
        <p className="text-2xs text-muted-foreground truncate max-w-[300px]">
          {stop.location ? formatLocation(stop.location) : "—"}
        </p>
      </div>
    </div>
  );
}
