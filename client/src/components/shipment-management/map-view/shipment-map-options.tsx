import { Checkbox } from "@/components/common/fields/checkbox";
import { Label } from "@/components/common/fields/label";
import {
  RadioGroup,
  RadioGroupItem,
} from "@/components/common/fields/radio-group";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { MapLayer, MapType, useShipmentMapStore } from "@/stores/ShipmentStore";
import { SlidersHorizontalIcon } from "lucide-react";

function MapOptionsButton() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button size="icon">
            <SlidersHorizontalIcon />
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">
          <p>Map Options</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function onMapBaseOptionsChange(value: MapType) {
  useShipmentMapStore.setState({ mapType: value });
}

function MapPopoverContent() {
  const mapType = useShipmentMapStore((state) => state.mapType);
  const [mapLayers, setMapLayers] = useShipmentMapStore((state) => [
    state.mapLayers,
    state.setMapLayers,
  ]);

  return (
    <div className="grid gap-2">
      <h4 className="text-sm font-semibold leading-none">Map Base</h4>
      <RadioGroup defaultValue="default" onValueChange={onMapBaseOptionsChange}>
        <div className="flex select-none items-center space-x-2">
          <RadioGroupItem
            value="roadmap"
            id="r1"
            checked={mapType === "roadmap"}
          />
          <Label htmlFor="r1">Default</Label>
        </div>
        <div className="flex select-none items-center space-x-2">
          <RadioGroupItem
            value="terrain"
            id="r2"
            checked={mapType === "terrain"}
          />
          <Label htmlFor="r2">Terrain</Label>
        </div>
        <div className="flex select-none items-center space-x-2">
          <RadioGroupItem
            value="hybrid"
            id="r3"
            checked={mapType === "hybrid"}
          />
          <Label htmlFor="r3">Satellite</Label>
        </div>
      </RadioGroup>
      <div className="grid gap-2">
        <h4 className="text-sm font-semibold leading-none">Overlay</h4>
        <div className="flex select-none items-center space-x-2">
          <Checkbox
            id="traffic"
            checked={mapLayers.includes("TrafficLayer")}
            onClick={() => {
              const newMapLayers = mapLayers.includes("TrafficLayer")
                ? mapLayers.filter((layer) => layer !== "TrafficLayer")
                : [...mapLayers, "TrafficLayer"];
              setMapLayers(newMapLayers as MapLayer[]);
            }}
          />
          <label
            htmlFor="traffic"
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            Traffic
          </label>
        </div>
        <div className="flex select-none items-center space-x-2">
          <Checkbox id="weather" disabled />
          <label
            htmlFor="weather"
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            Weather
          </label>
        </div>
        <div className="flex select-none items-center space-x-2">
          <Checkbox id="addresses" disabled />
          <label
            htmlFor="addresses"
            className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
          >
            Addresses
          </label>
        </div>
      </div>
    </div>
  );
}

export function ShipmentMapOptions() {
  return (
    <Popover>
      <PopoverTrigger>
        <MapOptionsButton />
      </PopoverTrigger>
      <PopoverContent className="w-auto" align="end">
        <MapPopoverContent />
      </PopoverContent>
    </Popover>
  );
}
