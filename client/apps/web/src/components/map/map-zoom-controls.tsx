import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { DEFAULT_ZOOM } from "@trenova/shared/lib/constants";
import { ControlPosition, MapControl, useMap } from "@vis.gl/react-google-maps";
import { MinusIcon, PlusIcon } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";

function ZoomButton({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={label}
            className="bg-background rounded-lg border backdrop-blur-sm"
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="left">{label}</TooltipContent>
    </Tooltip>
  );
}

export function MapZoomControls() {
  const t = useT();
  const map = useMap();

  const zoomBy = (step: number) => {
    if (!map) return;
    map.setZoom((map.getZoom() ?? DEFAULT_ZOOM) + step);
  };

  return (
    <MapControl position={ControlPosition.RIGHT_BOTTOM}>
      <div className="m-2.5 flex flex-col gap-1">
        <ZoomButton label={t("Zoom in")} onClick={() => zoomBy(1)}>
          <PlusIcon className="size-3.5" />
        </ZoomButton>
        <ZoomButton label={t("Zoom out")} onClick={() => zoomBy(-1)}>
          <MinusIcon className="size-3.5" />
        </ZoomButton>
      </div>
    </MapControl>
  );
}
