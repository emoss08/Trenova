/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
import { ShipmentMapAside } from "@/components/shipment-management/map-view/shipment-map-aside";
import { ShipmentMapOptions } from "@/components/shipment-management/map-view/shipment-map-options";
import { ShipmentMapZoom } from "@/components/shipment-management/map-view/shipment-map-zoom";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { GOOGLE_API_KEY } from "@/lib/constants";
import { useShipmentMapStore, useShipmentStore } from "@/stores/ShipmentStore";
import { GoogleMap } from "@google";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import { TooltipProvider } from "@radix-ui/react-tooltip";
import GoogleMapReact from "google-map-react";
import { useCallback } from "react";
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
          <div className="flex items-center justify-center w-4 h-4 bg-black rounded-full cursor-auto border-2 border-white" />
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

  const [mapType] = useShipmentMapStore.use("mapType");
  const [mapLayers] = useShipmentMapStore.use("mapLayers");
  const [map, setMap] = useShipmentMapStore.use("map");
  const [, setMaps] = useShipmentMapStore.use("maps");
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

  return (
    <>
      <div className="flex h-[700px] w-screen mx-auto space-x-10">
        <ShipmentMapAside />
        <div className="flex-grow relative">
          {/* Absolute positioned map options */}
          <div className="absolute top-0 right-0 z-10 p-2">
            <ShipmentMapOptions />
          </div>
          {/* Absolute positioned search field */}
          <div className="absolute top-0 left-0 z-10 p-2">
            <InputField
              name="searchMapQuery"
              control={control}
              placeholder="Search Shipments..."
              className="shadow-md pl-10"
              icon={
                <MagnifyingGlassIcon className="h-4 w-4 text-muted-foreground" />
              }
            />
          </div>
          {/* Absolute positioned zoom controls */}
          <div className="absolute bottom-0 right-0 z-10 p-2 mb-4">
            <ShipmentMapZoom map={map} />
          </div>
          {/* Google Map */}
          <GoogleMapReact
            bootstrapURLKeys={{ key: GOOGLE_API_KEY as string }}
            defaultCenter={defaultProps.center}
            defaultZoom={defaultProps.zoom}
            style={{ width: "100%", height: "100%" }}
            layerTypes={mapLayers}
            yesIWantToUseGoogleMapApiInternals
            onGoogleApiLoaded={handleApiLoaded}
            options={{
              mapTypeId: mapType,
              disableDefaultUI: true,
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
    </>
  );
}
