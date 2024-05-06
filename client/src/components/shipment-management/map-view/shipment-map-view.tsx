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

import { InputField } from "@/components/common/fields/input";
import { SendMessageDialog } from "@/components/common/send-message-dialog";
import { HourGridDialog } from "@/components/common/view-hos-logs";
import { MAP_STYLES } from "@/components/shipment-management/map-view/map-styles";
import { ShipmentMapAside } from "@/components/shipment-management/map-view/shipment-map-aside";
import { ShipmentMapOptions } from "@/components/shipment-management/map-view/shipment-map-options";
import { ShipmentMapZoom } from "@/components/shipment-management/map-view/shipment-map-zoom";
import { ComponentLoader } from "@/components/ui/component-loader";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useGoogleAPI } from "@/hooks/useQueries";
import { useShipmentMapStore, useShipmentStore } from "@/stores/ShipmentStore";
import { GoogleAPI } from "@/types/organization";
import { GoogleMap } from "@google";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { TooltipProvider } from "@radix-ui/react-tooltip";
import GoogleMapReact from "google-map-react";
import { useCallback, useState } from "react";
import { useForm } from "react-hook-form";

const markers = [
  { id: 1, lat: 37.78, lng: -122.41, label: "San Francisco" },
  { id: 2, lat: 34.05, lng: -118.24, label: "Los Angeles" },
  { id: 3, lat: 39.28, lng: -76.61, label: "Baltimore" },
  { id: 4, lat: 41.87, lng: -87.62, label: "Chicago" },
  { id: 5, lat: 40.71, lng: -74.01, label: "New York" },
  { id: 6, lat: 29.76, lng: -95.36, label: "Houston" },
  { id: 7, lat: 33.75, lng: -84.39, label: "Atlanta" },
  { id: 8, lat: 32.78, lng: -96.8, label: "Dallas" },
  { id: 9, lat: 39.95, lng: -75.16, label: "Philadelphia" },
  { id: 10, lat: 32.71, lng: -117.16, label: "San Diego" },
  { id: 11, lat: 32.79, lng: -79.94, label: "Charleston" },
  { id: 12, lat: 39.29, lng: -76.61, label: "Baltimore" },
  { id: 13, lat: 39.95, lng: -75.16, label: "Philadelphia" },
  { id: 14, lat: 37.78, lng: -122.41, label: "San Francisco" },
  { id: 15, lat: 34.05, lng: -118.24, label: "Los Angeles" },
  { id: 16, lat: 39.28, lng: -76.61, label: "Baltimore" },
  { id: 17, lat: 41.87, lng: -87.62, label: "Chicago" },
  { id: 18, lat: 40.71, lng: -74.01, label: "New York" },
  { id: 19, lat: 29.76, lng: -95.36, label: "Houston" },
  { id: 20, lat: 33.75, lng: -84.39, label: "Atlanta" },
  { id: 21, lat: 32.78, lng: -96.8, label: "Dallas" },
  { id: 22, lat: 39.95, lng: -75.16, label: "Philadelphia" },
  { id: 23, lat: 32.71, lng: -117.16, label: "San Diego" },
  { id: 24, lat: 32.79, lng: -79.94, label: "Charleston" },
  { id: 25, lat: 39.29, lng: -76.61, label: "Baltimore" },
  { id: 26, lat: 39.95, lng: -75.16, label: "Philadelphia" },
  { id: 27, lat: 37.78, lng: -122.41, label: "San Francisco" },
  { id: 28, lat: 34.05, lng: -118.24, label: "Los Angeles" },
];

interface MapMarkerProps {
  lat: number;
  lng: number;
  text: string;
}

function MapMarker({ text }: MapMarkerProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="relative size-4 -translate-x-1/2 -translate-y-1/2 cursor-auto items-center justify-center rounded-full border-2 border-white bg-black" />
        </TooltipTrigger>
        <TooltipContent>{text}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function ShipmentMapView() {
  const defaultProps = {
    center: {
      lat: 37.0902,
      lng: -95.7129,
    },
    zoom: 5,
  };
  const { control } = useForm();

  const mapType = useShipmentMapStore((state) => state.mapType);
  const mapLayers = useShipmentMapStore((state) => state.mapLayers);
  const [map, setMap] = useState<GoogleMap | null>(null);
  const [, setMaps] = useState<GoogleMap | null>(null);
  const [sendMessageDialogOpen, setSendMessageDialogOpen] =
    useShipmentStore.use("sendMessageDialogOpen");
  const [reviewLogDialogOpen, setReviewLogDialogOpen] = useShipmentStore.use(
    "reviewLogDialogOpen",
  );

  const handleApiLoaded = useCallback(
    ({ map, maps }: { map: GoogleMap; maps: GoogleMap }) => {
      setMap(map);
      setMaps(maps);
    },
    [],
  );

  // Get Google API Key
  const { googleAPIData, isLoading } = useGoogleAPI();
  const apiKey = (googleAPIData as GoogleAPI)?.apiKey as string;

  return isLoading ? (
    <div className="flex h-[50vh] w-screen items-center justify-center">
      <div className="flex flex-col items-center justify-center text-center">
        <ComponentLoader />
      </div>
    </div>
  ) : (
    <div className="mx-auto flex h-[700px] w-screen space-x-10">
      <ShipmentMapAside />
      <div className="relative w-full grow">
        <div className="absolute right-0 top-0 z-10 p-2">
          <ShipmentMapOptions />
        </div>
        <div className="absolute left-0 top-0 z-10 p-2">
          <InputField
            name="searchMapQuery"
            control={control}
            placeholder="Search Shipments..."
            className="pl-10 shadow-md"
            icon={
              <MagnifyingGlassIcon className="size-4 text-muted-foreground" />
            }
          />
        </div>
        <div className="absolute bottom-0 right-0 z-10 mb-4 p-2">
          <ShipmentMapZoom map={map} />
        </div>
        <GoogleMapReact
          bootstrapURLKeys={{ key: apiKey }}
          defaultCenter={defaultProps.center}
          defaultZoom={defaultProps.zoom}
          layerTypes={mapLayers}
          yesIWantToUseGoogleMapApiInternals
          onGoogleApiLoaded={handleApiLoaded}
          options={{
            mapTypeId: mapType,
            disableDefaultUI: true,
            styles: MAP_STYLES,
          }}
        >
          {markers.map((marker) => (
            <MapMarker
              key={marker.id}
              lat={marker.lat}
              lng={marker.lng}
              text={marker.label}
            />
          ))}
        </GoogleMapReact>
      </div>
      {sendMessageDialogOpen && (
        <SendMessageDialog
          open={sendMessageDialogOpen}
          onOpenChange={() => setSendMessageDialogOpen(false)}
        />
      )}
      {reviewLogDialogOpen && (
        <HourGridDialog
          open={reviewLogDialogOpen}
          onOpenChange={() => setReviewLogDialogOpen(false)}
        />
      )}
    </div>
  );
}
