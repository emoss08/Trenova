"use no memo";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { nanoid } from "nanoid";
import React from "react";
import type { UseFieldArrayRemove, UseFieldArrayUpdate } from "react-hook-form";
import { toast } from "sonner";
import { StopTimeline } from "../stop-details/stop-timeline-content";

function MoveListScrollArea({ children }: { children: React.ReactNode }) {
  return (
    <ScrollArea className="flex max-h-[250px] flex-col px-4 py-2 rounded-b-lg">
      {children}
    </ScrollArea>
  );
}

export function MoveListScrollAreaInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="relative py-4 space-y-6">{children}</div>;
}

export function MoveList({
  move,
  moveIdx,
}: {
  move: MoveSchema;
  moveIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
}) {
  "use no memo";
  return (
    <MoveListScrollArea>
      <MoveListScrollAreaInner>
        {move.stops.map((stop, stopIdx) => {
          if (!stop) {
            toast.error("Unable to render stop timeline", {
              description: "If this issue persists, please contact support.",
            });

            return (
              <div
                key={`stop-${moveIdx}-${stopIdx}`}
                className="p-4 text-destructive"
              >
                Failed to load stop timeline
              </div>
            );
          }

          const isLastStop = stopIdx === move.stops.length - 1;
          const nextStop = !isLastStop ? move.stops[stopIdx + 1] : null;
          const prevStopStatus =
            stopIdx > 0 ? move.stops[stopIdx - 1]?.status : undefined;

          const key = stop.id || `stop-${moveIdx}-${stopIdx}-${nanoid()}`;

          return (
            <StopTimeline
              key={key}
              stop={stop}
              nextStop={nextStop}
              isLast={isLastStop}
              moveStatus={move.status}
              moveIdx={moveIdx}
              stopIdx={stopIdx}
              prevStopStatus={prevStopStatus}
            />
          );
        })}
      </MoveListScrollAreaInner>
    </MoveListScrollArea>
  );
}
