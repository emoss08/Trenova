/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Icon } from "@/components/ui/icons";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import {
  locationSchema,
  type LocationSchema,
} from "@/lib/schemas/location-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { IntegrationType } from "@/types/integration";
import { faCopy } from "@fortawesome/pro-regular-svg-icons";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";
import { useForm } from "react-hook-form";
import { LocationForm } from "./location-form";

export function EditLocationModal({
  currentRecord,
}: EditTableSheetProps<LocationSchema>) {
  const form = useForm({
    resolver: zodResolver(locationSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      title="Location"
      formComponent={<LocationForm />}
      form={form}
      url="/locations/"
      queryKey="location-list"
      fieldKey="name"
      titleComponent={(currentRecord) => {
        return currentRecord ? (
          <div className="flex items-center gap-x-2">
            <span className="truncate max-w-[200px]">{currentRecord.name}</span>
            {currentRecord.isGeocoded ? (
              <GeocodedBadge location={currentRecord as LocationSchema} />
            ) : (
              <span
                title="Location is not geocoded"
                className="size-2 rounded-full bg-red-600"
              />
            )}
          </div>
        ) : null;
      }}
    />
  );
}
function GeocodedBadge({ location }: { location: LocationSchema }) {
  const { data: googleMapsData, isLoading } = useQuery({
    ...queries.integration.getIntegrationByType(IntegrationType.GoogleMaps),
    enabled: !!location.placeId,
  });

  const position = {
    lat: location.latitude || 0,
    lng: location.longitude || 0,
  };

  return (
    <HoverCard>
      <HoverCardTrigger asChild>
        <span className="size-2 rounded-full bg-purple-600" />
      </HoverCardTrigger>
      <HoverCardContent className="flex flex-col gap-2 p-2 w-auto">
        <div className="flex flex-col gap-0.5">
          <Row label="Longitude" value={location.longitude} />
          <Row label="Latitude" value={location.latitude} />
          <Row label="Place ID" value={location.placeId} />
        </div>
        {googleMapsData && !isLoading && (
          <div className="h-32 w-full rounded-md overflow-hidden border border-border">
            <APIProvider apiKey={googleMapsData.configuration.apiKey}>
              <Map
                defaultCenter={position}
                defaultZoom={17}
                mapId="DEMO_MAP_ID"
              >
                <AdvancedMarker position={position} />
              </Map>
            </APIProvider>
          </div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}

function Row({ label, value }: { label: string; value: any }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div
      className="group flex gap-4 text-sm justify-between items-center cursor-pointer"
      onClick={(e) => {
        e.stopPropagation();
        copy(value);
      }}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono truncate flex items-center gap-1">
        <span className="invisible group-hover:visible">
          {!isCopied ? (
            <Icon icon={faCopy} className="size-3" />
          ) : (
            <Icon icon={faCheck} className="size-3" />
          )}
        </span>
        {value}
      </span>
    </div>
  );
}
