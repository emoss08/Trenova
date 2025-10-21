/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ShipmentStatusBadge } from "@/components/status-badge";
import Highlight from "@/components/ui/highlight";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { LocationSchema } from "@/lib/schemas/location-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentLocations } from "@/lib/shipment/utils";
import { formatLocation } from "@/lib/utils";
import type { ShipmentCardProps } from "@/types/shipment";
import { faSignalStream } from "@fortawesome/pro-regular-svg-icons";
import { Timeline } from "./shipment-timeline";

export function ShipmentCard({
  shipment,
  onSelect,
  inputValue,
}: ShipmentCardProps) {
  const { origin } = ShipmentLocations.useLocations(shipment);

  if (!shipment) {
    return null;
  }

  const { status } = shipment;

  if (!origin) {
    return <p>-</p>;
  }

  return (
    <div className="p-2 border border-sidebar-border rounded-md bg-card text-sm">
      <div className="flex flex-col gap-2">
        <div className="flex justify-between w-full items-center">
          <ShipmentStatusBadge status={status} />
          <LocationGeocoded location={origin} />
        </div>
        <ProNumber
          shipment={shipment}
          onSelect={onSelect}
          inputValue={inputValue}
        />
        <StopInformation shipment={shipment} />
      </div>
    </div>
  );
}

function ProNumber({
  shipment,
  onSelect,
  inputValue,
}: {
  shipment: ShipmentSchema;
  onSelect: (shipmentId: string) => void;
  inputValue?: string;
}) {
  return (
    <div className="flex items-center gap-0.5">
      <button
        onClick={() => {
          onSelect(shipment.id ?? "");
        }}
        className="text-primary underline hover:text-primary/70 cursor-pointer"
      >
        <Highlight
          text={shipment.proNumber ?? ""}
          highlight={inputValue ?? ""}
        />
      </button>
    </div>
  );
}

function StopInformation({ shipment }: { shipment: ShipmentSchema }) {
  const { destination, origin } = ShipmentLocations.useLocations(shipment);

  if (!origin || !destination) {
    return <p>-</p>;
  }

  const items = [
    {
      id: "location-1",
      content: (
        <div className="rounded-lg">
          <p className="text-2xs text-muted-foreground">
            {formatLocation(origin)}
          </p>
        </div>
      ),
    },
    {
      id: "location-2",
      content: (
        <div className="rounded-lg">
          <p className="text-2xs text-muted-foreground">
            {formatLocation(destination)}
          </p>
        </div>
      ),
    },
  ];

  return <Timeline items={items} />;
}

export function LocationGeocoded({ location }: { location: LocationSchema }) {
  return !location.isGeocoded ? (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="relative flex size-4">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-600 opacity-75"></span>
            <Icon icon={faSignalStream} className="size-4 text-red-500" />
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <p>Origin Location Not Geocoded</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  ) : null;
}
