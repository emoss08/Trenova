import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopSchema } from "@/lib/schemas/stop-schema";
import { formatLocation } from "@/lib/utils";
import { useMemo } from "react";
export function ShipmentMovePreview({
  moves,
}: {
  moves: ShipmentSchema["moves"];
}) {
  const stopsOrderedBySequence = useMemo(() => {
    return moves
      .flatMap((move) => move.stops)
      .sort((a, b) => a.sequence - b.sequence);
  }, [moves]);

  return (
    <div className="flex flex-col gap-2">
      {moves.map((move) => (
        <div key={move.id} className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            Move {move.sequence + 1}
          </div>
          <div className="flex flex-col gap-1">
            {stopsOrderedBySequence.map((stop) => (
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
    <div className="flex items-center justify-between gap-2 truncate">
      <div className="text-2xs text-muted-foreground">
        {stop.location ? formatLocation(stop.location) : "—"}
      </div>
    </div>
  );
}
