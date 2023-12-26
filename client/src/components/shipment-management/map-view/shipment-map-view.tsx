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
import { GOOGLE_API_KEY } from "@/lib/constants";
import { MagnifyingGlassIcon } from "@radix-ui/react-icons";
import GoogleMapReact from "google-map-react";
import { useForm } from "react-hook-form";
import { InputField } from "@/components/common/fields/input";
import { ShipmentMapOptions } from "@/components/shipment-management/map-view/shipment-map-options";
import { useShipmentMapStore } from "@/stores/ShipmentStore";
import { ShipmentMapAside } from "@/components/shipment-management/map-view/shipment-map-aside";
import { ShipmentMapZoom } from "@/components/shipment-management/map-view/shipment-map-zoom";
import { useCallback } from "react";
import { GoogleMap } from "@google";

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

  const handleApiLoaded = useCallback(
    ({ map, maps }: { map: GoogleMap; maps: GoogleMap }) => {
      setMap(map);
      setMaps(maps);
    },
    [],
  );

  return (
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
            placeholder="Search Orders..."
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
          style={{ width: "100%", height: "100%" }} // Ensure the map fills the container
          layerTypes={mapLayers}
          yesIWantToUseGoogleMapApiInternals
          onGoogleApiLoaded={handleApiLoaded}
          options={{
            mapTypeId: mapType,
            disableDefaultUI: true,
          }}
        />
      </div>
    </div>
  );
}
