import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { MapStyleId, OverlayId } from "@/types/shipment-map";
import { ControlPosition, MapControl, useMap } from "@vis.gl/react-google-maps";
import { LocateFixedIcon, Maximize2Icon, Minimize2Icon } from "lucide-react";
import { useCallback } from "react";
import type { MapFilters } from "./map-filter-bar";
import { MapFilterPopover } from "./map-filter-popover";
import { MapLegendPopover } from "./map-legend-popover";
import { MapOptionsPopover } from "./map-options-popover";
import type { MockTractor } from "./mock-data";

export function MapControls({
  mapStyle,
  onMapStyleChange,
  overlays,
  onToggleOverlay,
  isFullscreen,
  onToggleFullscreen,
  filteredTractors,
  filters,
  onFiltersChange,
  totalCount,
}: {
  mapStyle: MapStyleId;
  onMapStyleChange: (s: MapStyleId) => void;
  overlays: Record<OverlayId, boolean>;
  onToggleOverlay: (id: OverlayId) => void;
  isFullscreen: boolean;
  onToggleFullscreen: () => void;
  filteredTractors: MockTractor[];
  filters: MapFilters;
  onFiltersChange: (filters: MapFilters) => void;
  totalCount: number;
}) {
  const map = useMap();

  const handleZoomToFit = useCallback(() => {
    if (!map || filteredTractors.length === 0) return;

    const bounds = new google.maps.LatLngBounds();

    for (const tractor of filteredTractors) {
      bounds.extend({ lat: tractor.lat, lng: tractor.lng });

      if (overlays.routes) {
        for (const point of tractor.routePath) {
          bounds.extend(point);
        }
      }

      if (overlays.stops) {
        for (const stop of tractor.stops) {
          bounds.extend({ lat: stop.lat, lng: stop.lng });
        }
      }
    }

    map.fitBounds(bounds, { top: 48, right: 48, bottom: 48, left: 48 });
  }, [map, filteredTractors, overlays.routes, overlays.stops]);

  return (
    <MapControl position={ControlPosition.RIGHT_TOP}>
      <div className="m-2.5 flex flex-col gap-1">
        <MapOptionsPopover
          mapStyle={mapStyle}
          onMapStyleChange={onMapStyleChange}
          overlays={overlays}
          onToggleOverlay={onToggleOverlay}
        />
        <MapFilterPopover
          filters={filters}
          onFiltersChange={onFiltersChange}
          totalCount={totalCount}
          filteredCount={filteredTractors.length}
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
