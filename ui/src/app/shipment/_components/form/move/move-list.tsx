import { LazyComponent } from "@/components/error-boundary";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { ShipmentMove } from "@/types/move";
import { nanoid } from "nanoid";
import React, { lazy } from "react";
import type { UseFieldArrayRemove, UseFieldArrayUpdate } from "react-hook-form";
import { toast } from "sonner";

const StopTimeline = lazy(
  () =>
    import(
      "@/app/shipment/_components/sidebar/stop-details/stop-timeline-content"
    ),
);

function MoveListScrollArea({ children }: { children: React.ReactNode }) {
  return (
    <ScrollArea className="flex max-h-[250px] flex-col overflow-y-auto px-4 py-2 rounded-b-lg">
      {children}
      <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-8 bg-gradient-to-t from-background to-transparent z-50" />
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
  update,
  remove,
}: {
  move: ShipmentMove;
  moveIdx: number;
  update: UseFieldArrayUpdate<ShipmentSchema, "moves">;
  remove: UseFieldArrayRemove;
}) {
  return (
    <MoveListScrollArea>
      <MoveListScrollAreaInner>
        {move.stops.map((stop, stopIdx) => {
          if (!stop) {
            toast.error("Unable to render stop timeline", {
              description: "If this issue persists, please contact support.",
            });

            throw new Error("Unable to render stop timeline");
          }

          const isLastStop = stopIdx === move.stops.length - 1;
          const nextStop = !isLastStop ? move.stops[stopIdx + 1] : null;
          const prevStopStatus =
            stopIdx > 0 ? move.stops[stopIdx - 1]?.status : undefined;

          const key = stop.id || `stop-${moveIdx}-${stopIdx}-${nanoid()}`;

          return (
            <LazyComponent key={key}>
              <StopTimeline
                stop={stop}
                nextStop={nextStop}
                isLast={isLastStop}
                moveStatus={move.status}
                moveIdx={moveIdx}
                stopIdx={stopIdx}
                update={update}
                remove={remove}
                prevStopStatus={prevStopStatus}
              />
            </LazyComponent>
          );
        })}
      </MoveListScrollAreaInner>
    </MoveListScrollArea>
  );
}
