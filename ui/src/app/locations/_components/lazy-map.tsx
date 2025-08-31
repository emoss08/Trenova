/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { AdvancedMarker, APIProvider, Map } from "@vis.gl/react-google-maps";

interface LazyMapProps {
  apiKey: string;
  position: { lat: number; lng: number };
}

export default function LazyMap({ apiKey, position }: LazyMapProps) {
  return (
    <APIProvider apiKey={apiKey}>
      <Map defaultCenter={position} defaultZoom={17} mapId="DEMO_MAP_ID">
        <AdvancedMarker position={position} />
      </Map>
    </APIProvider>
  );
}
