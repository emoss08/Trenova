import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  CircleDotIcon,
  CloudSunIcon,
  LayersIcon,
  MapPinIcon,
  RouteIcon,
  TrafficConeIcon,
  TruckIcon,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { MapStyleId, OverlayId } from "./use-map-ui-state";

type OverlayConfig = {
  id: OverlayId;
  label: string;
  icon: LucideIcon;
};

const OVERLAY_OPTIONS: OverlayConfig[] = [
  { id: "traffic", label: "Traffic", icon: TrafficConeIcon },
  { id: "weather", label: "Weather", icon: CloudSunIcon },
  { id: "vehicles", label: "Vehicles", icon: TruckIcon },
  { id: "routes", label: "Routes", icon: RouteIcon },
  { id: "stops", label: "Stops", icon: MapPinIcon },
  { id: "geofences", label: "Geofences", icon: CircleDotIcon },
];

const MAP_BASE_OPTIONS: { id: MapStyleId; label: string }[] = [
  { id: "roadmap", label: "Default" },
  { id: "terrain", label: "Terrain" },
  { id: "satellite", label: "Satellite" },
  { id: "hybrid", label: "Hybrid" },
];

export function MapOptionsPopover({
  mapStyle,
  onMapStyleChange,
  overlays,
  onToggleOverlay,
}: {
  mapStyle: MapStyleId;
  onMapStyleChange: (s: MapStyleId) => void;
  overlays: Record<OverlayId, boolean>;
  onToggleOverlay: (id: OverlayId) => void;
}) {
  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <Button
                  variant="outline"
                  size="icon"
                  className="size-8 bg-background shadow-sm"
                />
              }
            />
          }
        >
          <LayersIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent side="left">Map options</TooltipContent>
      </Tooltip>
      <PopoverContent side="left" sideOffset={8} className="w-48 p-0">
        <div className="max-h-[70vh] overflow-y-auto p-3">
          <SectionLabel>Map Base</SectionLabel>
          <div className="mt-1.5 flex flex-col">
            {MAP_BASE_OPTIONS.map((opt) => (
              <label
                key={opt.id}
                className="flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-accent"
              >
                <input
                  type="radio"
                  name="map-base"
                  checked={mapStyle === opt.id}
                  onChange={() => onMapStyleChange(opt.id)}
                  className="accent-brand"
                />
                {opt.label}
              </label>
            ))}
          </div>

          <Separator className="my-2.5" />

          <SectionLabel>Overlay</SectionLabel>
          <div className="mt-1.5 flex flex-col">
            {OVERLAY_OPTIONS.map((opt) => (
              <label
                key={opt.id}
                className="flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-accent"
              >
                <Checkbox
                  checked={overlays[opt.id]}
                  onCheckedChange={() => onToggleOverlay(opt.id)}
                />
                <opt.icon className="size-3.5 text-muted-foreground" />
                {opt.label}
              </label>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </span>
  );
}
