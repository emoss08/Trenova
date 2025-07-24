/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { LazyComponent } from "@/components/error-boundary";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { MoveSchema } from "@/lib/schemas/move-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
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
}: {
  move: MoveSchema;
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
            <LazyComponent key={key}>
              <StopTimeline
                stop={stop}
                nextStop={nextStop}
                isLast={isLastStop}
                moveStatus={move.status}
                moveIdx={moveIdx}
                stopIdx={stopIdx}
                prevStopStatus={prevStopStatus}
              />
            </LazyComponent>
          );
        })}
      </MoveListScrollAreaInner>
    </MoveListScrollArea>
  );
}
