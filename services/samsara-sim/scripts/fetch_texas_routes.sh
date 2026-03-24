#!/usr/bin/env bash

set -euo pipefail

OUTPUT_PATH="${1:-./config/datasets/texas_osm_routes.geojson}"
mkdir -p "$(dirname "${OUTPUT_PATH}")"

fetch_feature() {
  local route_name="$1"
  local asset_id="$2"
  local speed_mps="$3"
  local coordinates="$4"
  local source_url="https://router.project-osrm.org/route/v1/driving/${coordinates}?overview=full&geometries=geojson&steps=false"

  curl -fsSL "${source_url}" | jq -c \
    --arg route_name "${route_name}" \
    --arg asset_id "${asset_id}" \
    --argjson speed_mps "${speed_mps}" '
      .routes[0].geometry as $geometry
      | if $geometry == null then empty else
        {
          type: "Feature",
          properties: {
            name: $route_name,
            assetId: $asset_id,
            speedMps: $speed_mps,
            source: "OSRM demo server (OpenStreetMap network)"
          },
          geometry: {
            type: "LineString",
            coordinates: $geometry.coordinates
          }
        }
      end
    '
}

features=()
features+=("$(fetch_feature "Austin Roundtrip" "veh-1001" 20 "-97.7431,30.2672;-97.8347,30.5145;-97.7019,30.6550;-97.5164,30.3074;-97.7431,30.2672")")
features+=("$(fetch_feature "Dallas Metro Arc" "veh-1002" 19 "-96.7970,32.7767;-96.9910,32.9254;-97.3327,32.7555;-96.7970,32.7767")")
features+=("$(fetch_feature "Houston Belt Sweep" "veh-1003" 21 "-95.3698,29.7604;-95.4611,29.9941;-95.1460,30.0400;-95.0200,29.7700;-95.3698,29.7604")")
features+=("$(fetch_feature "San Antonio To Austin Loop" "veh-1004" 20 "-98.4936,29.4241;-98.1256,29.8833;-97.7431,30.2672;-98.4936,29.4241")")
features+=("$(fetch_feature "Fort Worth Waco Corridor" "veh-1005" 20 "-97.3308,32.7555;-97.1467,31.5493;-97.3308,32.7555")")
features+=("$(fetch_feature "El Paso Long Eastbound" "veh-1006" 24 "-106.4850,31.7619;-105.9650,31.3500;-105.5100,31.6900;-105.1800,31.7500")")
features+=("$(fetch_feature "Lubbock Abilene Run" "veh-1007" 22 "-101.8552,33.5779;-100.0000,33.1000;-99.7331,32.4487;-101.8552,33.5779")")
features+=("$(fetch_feature "Corpus SA Shuttle" "veh-1008" 21 "-97.3964,27.8006;-98.1556,28.8053;-98.4936,29.4241;-97.3964,27.8006")")
features+=("$(fetch_feature "Tyler Dallas Commute" "veh-1009" 20 "-95.3011,32.3513;-95.6000,32.5213;-96.7970,32.7767;-95.3011,32.3513")")
features+=("$(fetch_feature "Midland Odessa Big Spring" "veh-1010" 23 "-102.0779,31.9974;-102.3676,31.8457;-101.4787,32.2504;-102.0779,31.9974")")
features+=("$(fetch_feature "Amarillo Wichita Falls" "veh-1011" 22 "-101.8313,35.2219;-100.2000,34.8000;-98.4934,33.9137;-101.8313,35.2219")")
features+=("$(fetch_feature "Brownsville Coastal North" "veh-1012" 21 "-97.4975,25.9017;-97.8550,26.2034;-97.3964,27.8006;-97.4975,25.9017")")

printf "%s\n" "${features[@]}" | jq -s '{type:"FeatureCollection",features:.}' > "${OUTPUT_PATH}"
echo "wrote route dataset: ${OUTPUT_PATH}"
