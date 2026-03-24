import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { AdvancedMarker } from "@vis.gl/react-google-maps";

import type { MockTractor } from "./mock-data";

export function TractorMarker({
  tractor,
  isSelected,
  onClick,
}: {
  tractor: MockTractor;
  isSelected: boolean;
  onClick: (id: string) => void;
}) {
  const isDelayed = tractor.shipmentStatus === "Delayed";

  return (
    <AdvancedMarker
      position={{ lat: tractor.lat, lng: tractor.lng }}
      onClick={() => onClick(tractor.id)}
      zIndex={isSelected ? 100 : isDelayed ? 50 : 1}
    >
      <Tooltip>
        <TooltipTrigger
          render={
            <div className="relative flex cursor-pointer items-center justify-center" />
          }
        >
          {isDelayed && (
            <span
              className="absolute size-7 animate-ping rounded-full bg-destructive/50"
              aria-hidden
            />
          )}
          <div
            className={`relative size-5 rounded-full ${
              isSelected ? "bg-brand" : "bg-black"
            }`}
            style={{
              border: "2px solid white",
              boxShadow: "0 1px 4px rgba(0,0,0,0.5)",
            }}
          />
        </TooltipTrigger>
        <TooltipContent side="top" sideOffset={8}>
          {tractor.unitNumber} — {tractor.shipmentStatus}
        </TooltipContent>
      </Tooltip>
    </AdvancedMarker>
  );
}
