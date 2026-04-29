import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { MapStyleId, OverlayId } from "@/types/shipment-map";
import { ControlPosition, MapControl, useMap } from "@vis.gl/react-google-maps";
import { LocateFixedIcon, Maximize2Icon, Minimize2Icon } from "lucide-react";
import { useCallback } from "react";
import { MapLegendPopover } from "./map-legend-popover";
import { MapOptionsPopover } from "./map-options-popover";

export function MapControls({
  mapStyle,
  onMapStyleChange,
  overlays,
  onToggleOverlay,
  isFullscreen,
  onToggleFullscreen,
  boundsPoints,
}: {
  mapStyle: MapStyleId;
  onMapStyleChange: (s: MapStyleId) => void;
  overlays: Record<OverlayId, boolean>;
  onToggleOverlay: (id: OverlayId) => void;
  isFullscreen: boolean;
  onToggleFullscreen: () => void;
  boundsPoints: google.maps.LatLngLiteral[];
}) {
  const map = useMap();

  const handleZoomToFit = useCallback(() => {
    if (!map || boundsPoints.length === 0) return;
    const bounds = new google.maps.LatLngBounds();
    for (const point of boundsPoints) bounds.extend(point);
    map.fitBounds(bounds, { top: 48, right: 48, bottom: 48, left: 48 });
  }, [map, boundsPoints]);

  return (
    <MapControl position={ControlPosition.RIGHT_TOP}>
      <div className="m-2.5 flex flex-col gap-1">
        <MapOptionsPopover
          mapStyle={mapStyle}
          onMapStyleChange={onMapStyleChange}
          overlays={overlays}
          onToggleOverlay={onToggleOverlay}
        />
        <MapLegendPopover />
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                className="size-8 bg-background shadow-sm"
                onClick={onToggleFullscreen}
              />
            }
          >
            {isFullscreen ? (
              <Minimize2Icon className="size-4" />
            ) : (
              <Maximize2Icon className="size-4" />
            )}
          </TooltipTrigger>
          <TooltipContent side="left">
            {isFullscreen ? "Exit fullscreen" : "Fullscreen"}
          </TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="outline"
                size="icon"
                className="size-8 bg-background shadow-sm"
                onClick={handleZoomToFit}
                disabled={boundsPoints.length === 0}
              />
            }
          >
            <LocateFixedIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent side="left">Zoom to fit</TooltipContent>
        </Tooltip>
      </div>
    </MapControl>
  );
}
