"use no memo";
import { useMap } from "@vis.gl/react-google-maps";
import { useEffect, useRef } from "react";

export function FitBounds({
  points,
}: {
  points: { lat: number; lng: number }[];
}) {
  const map = useMap();
  const pts = points;
  useEffect(() => {
    if (!map || pts.length === 0 || !(window as any).google) return;
    const bounds = new google.maps.LatLngBounds();
    pts.forEach((p) => bounds.extend(p));
    try {
      map.fitBounds(bounds, { top: 24, right: 24, bottom: 24, left: 24 });
    } catch {
      map.fitBounds(bounds);
    }
  }, [map, pts]);
  return null;
}

export function DirectionsPolyline({
  points,
}: {
  points: { lat: number; lng: number }[];
}) {
  const map = useMap();
  const polylineRef = useRef<google.maps.Polyline | null>(null);
  const requestIdRef = useRef(0);

  useEffect(() => {
    if (!map || points.length < 2 || !(window as any).google) return;

    // Bump request id to invalidate any in-flight responses
    const myRequestId = ++requestIdRef.current;

    // Clear any existing polyline immediately
    if (polylineRef.current) {
      polylineRef.current.setMap(null);
      polylineRef.current = null;
    }

    const origin = points[0];
    const destination = points[points.length - 1];
    const waypoints = points
      .slice(1, -1)
      .map((p) => ({ location: p, stopover: true }));

    const directionsService = new google.maps.DirectionsService();

    directionsService.route(
      {
        origin,
        destination,
        waypoints,
        travelMode: google.maps.TravelMode.DRIVING,
        optimizeWaypoints: false,
        avoidFerries: false,
        avoidHighways: false,
        avoidTolls: false,
        provideRouteAlternatives: false,
      },
      (result, status) => {
        // Ignore stale responses
        if (requestIdRef.current !== myRequestId) return;
        if (status === google.maps.DirectionsStatus.OK && result) {
          const route = result.routes[0];
          const path = route.overview_path;
          // Clear again just in case
          if (polylineRef.current) polylineRef.current.setMap(null);
          polylineRef.current = new google.maps.Polyline({
            map,
            path,
            strokeColor: "#2563eb",
            strokeOpacity: 0.9,
            strokeWeight: 3,
          });
        }
      },
    );

    return () => {
      // Invalidate this effect's result using captured id
      requestIdRef.current = myRequestId + 1;
      if (polylineRef.current) {
        polylineRef.current.setMap(null);
        polylineRef.current = null;
      }
    };
  }, [map, points]);
  return null;
}
