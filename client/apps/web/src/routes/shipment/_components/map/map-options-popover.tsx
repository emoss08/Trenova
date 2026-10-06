import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type { MapStyleId, OverlayId } from "@/types/shipment-map";
import {
  AlertTriangleIcon,
  CircleDotIcon,
  CloudSun02Icon,
  type IconComponent,
  LayersThree01Icon,
  MarkerPin04Icon,
  TrafficConeIcon,
  Truck01Icon,
} from "@trenova/shared/components/icons";

type OverlayConfig = {
  id: OverlayId;
  label: string;
  icon: IconComponent;
};

const OVERLAY_OPTIONS: OverlayConfig[] = [
  { id: "vehicles", label: "Live vehicles", icon: Truck01Icon },
  { id: "geofences", label: "Geofences", icon: CircleDotIcon },
  { id: "addresses", label: "Addresses", icon: MarkerPin04Icon },
  { id: "traffic", label: "Traffic", icon: TrafficConeIcon },
  { id: "weather", label: "Weather", icon: CloudSun02Icon },
  { id: "alerts", label: "Weather alerts", icon: AlertTriangleIcon },
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
  const t = useT();

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={<Button variant="outline" size="icon" className="bg-background size-7" />}
            />
          }
        >
          <LayersThree01Icon className="size-4" />
        </TooltipTrigger>
        <TooltipContent side="bottom">{t("Map options")}</TooltipContent>
      </Tooltip>
      <PopoverContent side="bottom" sideOffset={8} className="w-48 p-0">
        <div className="max-h-[70vh] overflow-y-auto p-3">
          <SectionLabel>{t("Map base")}</SectionLabel>
          <div className="mt-1.5 flex flex-col">
            {MAP_BASE_OPTIONS.map((opt) => (
              <label
                key={opt.id}
                className="hover:bg-accent flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-sm"
              >
                <input
                  type="radio"
                  name="map-base"
                  checked={mapStyle === opt.id}
                  onChange={() => onMapStyleChange(opt.id)}
                  className="accent-brand"
                />
                {t(opt.label)}
              </label>
            ))}
          </div>
          <Separator className="my-2.5" />
          <SectionLabel>{t("Overlay")}</SectionLabel>
          <div className="mt-1.5 flex flex-col">
            {OVERLAY_OPTIONS.map((opt) => (
              <label
                key={opt.id}
                className="hover:bg-accent flex cursor-pointer items-center gap-2 rounded-md px-1.5 py-1 text-sm"
              >
                <Checkbox
                  checked={overlays[opt.id]}
                  onCheckedChange={() => onToggleOverlay(opt.id)}
                />
                <opt.icon className="text-muted-foreground size-3.5" />
                {t(opt.label)}
              </label>
            ))}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <span className="text-muted-foreground text-xs font-semibold">{children}</span>;
}
