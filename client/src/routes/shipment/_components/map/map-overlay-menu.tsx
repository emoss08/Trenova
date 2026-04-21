import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { OverlayId, WeatherLayerId } from "@/types/shipment-map";
import type { LucideIcon } from "lucide-react";
import {
  CircleDotIcon,
  CloudIcon,
  CloudRainIcon,
  GaugeIcon,
  MapPinIcon,
  RouteIcon,
  SlidersHorizontalIcon,
  ThermometerIcon,
  TrafficConeIcon,
  TruckIcon,
  WindIcon,
  XIcon,
} from "lucide-react";

type OverlayConfig = {
  id: OverlayId;
  label: string;
  icon: LucideIcon;
};

const FLEET_OVERLAYS: OverlayConfig[] = [
  { id: "vehicles", label: "Vehicle Markers", icon: TruckIcon },
  { id: "routes", label: "Route Polylines", icon: RouteIcon },
  { id: "stops", label: "Stop Markers", icon: MapPinIcon },
  { id: "geofences", label: "Geofences", icon: CircleDotIcon },
];

type WeatherOption = {
  id: WeatherLayerId;
  label: string;
  icon: LucideIcon;
  requiresOWM?: boolean;
};

const WEATHER_OPTIONS: WeatherOption[] = [
  { id: "none", label: "None", icon: XIcon },
  { id: "precipitation", label: "Precipitation Radar", icon: CloudRainIcon },
  { id: "wind", label: "Wind Speed", icon: WindIcon, requiresOWM: true },
  { id: "clouds", label: "Cloud Cover", icon: CloudIcon, requiresOWM: true },
  { id: "temperature", label: "Temperature", icon: ThermometerIcon, requiresOWM: true },
  { id: "pressure", label: "Pressure", icon: GaugeIcon, requiresOWM: true },
];

export function MapOverlayMenu({
  overlays,
  onToggleOverlay,
  weatherLayer,
  onWeatherLayerChange,
  owmConfigured,
}: {
  overlays: Record<OverlayId, boolean>;
  onToggleOverlay: (id: OverlayId) => void;
  weatherLayer: WeatherLayerId;
  onWeatherLayerChange: (layer: WeatherLayerId) => void;
  owmConfigured: boolean;
}) {
  return (
    <DropdownMenu>
      <Tooltip>
        <TooltipTrigger
          render={<DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm" />} />}
        >
          <SlidersHorizontalIcon className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent side="bottom">Map overlays</TooltipContent>
      </Tooltip>
      <DropdownMenuContent align="end" sideOffset={6} className="w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel>Fleet</DropdownMenuLabel>
          {FLEET_OVERLAYS.map((item) => (
            <DropdownMenuCheckboxItem
              key={item.id}
              checked={overlays[item.id]}
              onCheckedChange={() => onToggleOverlay(item.id)}
            >
              <item.icon className="size-3.5 text-muted-foreground" />
              <span>{item.label}</span>
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel>Traffic</DropdownMenuLabel>
          <DropdownMenuCheckboxItem
            checked={overlays.traffic}
            onCheckedChange={() => onToggleOverlay("traffic")}
          >
            <TrafficConeIcon className="size-3.5 text-muted-foreground" />
            <span>Traffic Layer</span>
          </DropdownMenuCheckboxItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel>Weather</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={weatherLayer}
            onValueChange={(val) => onWeatherLayerChange(val as WeatherLayerId)}
          >
            {WEATHER_OPTIONS.map((opt) => {
              const needsConfig = opt.requiresOWM && !owmConfigured;
              return (
                <DropdownMenuRadioItem key={opt.id} value={opt.id} disabled={needsConfig}>
                  <opt.icon className="size-3.5 text-muted-foreground" />
                  <span>{opt.label}</span>
                  {needsConfig && (
                    <span className="ml-auto text-2xs text-muted-foreground">(Setup Required)</span>
                  )}
                </DropdownMenuRadioItem>
              );
            })}
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
