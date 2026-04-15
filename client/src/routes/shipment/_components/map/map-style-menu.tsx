import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { MapStyleId } from "@/types/shipment-map";
import { LayersIcon, MapIcon, MountainIcon, SatelliteDishIcon } from "lucide-react";

const MAP_STYLES: { value: MapStyleId; label: string; icon: React.ReactNode }[] = [
  { value: "roadmap", label: "Roadmap", icon: <MapIcon className="size-3.5" /> },
  { value: "satellite", label: "Satellite", icon: <SatelliteDishIcon className="size-3.5" /> },
  { value: "hybrid", label: "Hybrid", icon: <LayersIcon className="size-3.5" /> },
  { value: "terrain", label: "Terrain", icon: <MountainIcon className="size-3.5" /> },
];

export function MapStyleMenu({
  mapStyle,
  onMapStyleChange,
}: {
  mapStyle: MapStyleId;
  onMapStyleChange: (style: MapStyleId) => void;
}) {
  return (
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger
          render={<DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />} />}
        >
          <MapIcon className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent side="bottom">Map style</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" sideOffset={6}>
        <DropdownMenuGroup>
          <DropdownMenuLabel>Map Style</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={mapStyle}
            onValueChange={(val) => onMapStyleChange(val as MapStyleId)}
          >
            {MAP_STYLES.map((style) => (
              <DropdownMenuRadioItem key={style.value} value={style.value}>
                {style.icon}
                {style.label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
