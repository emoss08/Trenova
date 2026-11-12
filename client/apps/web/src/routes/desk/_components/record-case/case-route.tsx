import type { DeskCaseRoute } from "@/lib/graphql/desk-case";
import { lazyComponent } from "@/lib/lazy-component";
import { integration } from "@/lib/queries/integration";
import { useQuery } from "@tanstack/react-query";
import { Suspense, useState } from "react";
import { ErrorBoundary } from "react-error-boundary";
import { RouteSketch } from "./route-sketch";

const RouteMap = lazyComponent(
  () => import("./route-map"),
  (module) => module.default,
);

/** Whether every stop, and the truck when there is one, came with a place on the earth. */
export function routeHasCoordinates(route: DeskCaseRoute): boolean {
  const placed = (at: { latitude: number; longitude: number }) =>
    Number.isFinite(at.latitude) &&
    Number.isFinite(at.longitude) &&
    (at.latitude !== 0 || at.longitude !== 0);

  return (
    route.points.length > 0 && route.points.every(placed) && (!route.truck || placed(route.truck))
  );
}

/**
 * Where the load is. On Google Maps when the organization has a key and the
 * server sent each stop's place; otherwise the sketch, which needs neither.
 * The sketch is also what stands in while the map's code is read, and what
 * the exhibit falls back to whenever the map cannot be drawn: Google refuses
 * the key, the key is out of quota, the script does not load, or the map
 * throws while drawing. A map that will not draw is not an error the person
 * can act on from a case, so they are shown the route either way.
 */
export function CaseRoute({ route }: { route: DeskCaseRoute }) {
  const mappable = routeHasCoordinates(route);
  // The same query the integration's own form invalidates when the key is
  // saved, so a key added there shows on the next case without a reload.
  const key = useQuery({ ...integration.runtimeConfig("GoogleMaps"), enabled: mappable });
  const apiKey = key.data?.config.apiKey ?? "";
  // The key Google turned down, so a key saved since is tried afresh.
  const [refused, setRefused] = useState<string | null>(null);
  const sketch = <RouteSketch route={route} />;

  if (!mappable || apiKey === "" || refused === apiKey) {
    return sketch;
  }

  return (
    <ErrorBoundary fallback={sketch} resetKeys={[apiKey]}>
      <Suspense fallback={sketch}>
        <RouteMap.Component
          route={route}
          apiKey={apiKey}
          onUnavailable={() => setRefused(apiKey)}
        />
      </Suspense>
    </ErrorBoundary>
  );
}
