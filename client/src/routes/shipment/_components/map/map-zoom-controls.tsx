import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DEFAULT_ZOOM } from "@/lib/constants";
import { ControlPosition, MapControl, useMap } from "@vis.gl/react-google-maps";
import { MinusIcon, PlusIcon } from "lucide-react";
import { useCallback } from "react";

export function MapZoomControls() {
  const map = useMap();

  const zoomIn = useCallback(() => {
    if (!map) return;
    map.setZoom((map.getZoom() ?? DEFAULT_ZOOM) + 1);
  }, [map]);

  const zoomOut = useCallback(() => {
    if (!map) return;
    map.setZoom((map.getZoom() ?? DEFAULT_ZOOM) - 1);
  }, [map]);

  return (
    <MapControl position={ControlPosition.RIGHT_BOTTOM}>
      <div className="m-2.5 flex flex-col gap-1">
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="rounded-lg border bg-background/95 shadow-sm backdrop-blur-sm"
                onClick={zoomIn}
              />
            }
          >
            <PlusIcon className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent side="left">Zoom in</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="rounded-lg border bg-background/95 shadow-sm backdrop-blur-sm"
                onClick={zoomOut}
              />
            }
          >
            <MinusIcon className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent side="left">Zoom out</TooltipContent>
        </Tooltip>
      </div>
    </MapControl>
  );
}
