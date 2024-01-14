/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

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
import { ComponentIcon } from "lucide-react";

function MapOptionsButton() {
  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className="bg-background text-foreground hover:text-background"
            size="icon"
          >
            <ComponentIcon size={24} />
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
    <div className="grid gap-4">
      <div className="space-y-2">
        <h4 className="text-sm font-semibold leading-none">Map Base</h4>
        <RadioGroup
          defaultValue="default"
          onValueChange={onMapBaseOptionsChange}
        >
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
      </div>
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
