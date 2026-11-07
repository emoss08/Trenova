import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { integration } from "@/lib/queries/integration";
import { useQuery } from "@tanstack/react-query";
import { ColorScheme } from "@vis.gl/react-google-maps";

/** Google's own map style, which draws advanced markers when a tenant has not set a Map ID. */
const DEMO_MAP_ID = "DEMO_MAP_ID";

/**
 * The tenant's Map ID, from its Google Maps integration (a Map ID works only with keys from
 * its own Cloud project), and the colour scheme that picks its light or dark style.
 */
export function useMapStyle(): { mapId: string; colorScheme: ColorScheme } {
  const theme = useResolvedTheme();
  const { data } = useQuery(integration.runtimeConfig("GoogleMaps"));

  return {
    mapId: data?.config.mapId || DEMO_MAP_ID,
    colorScheme: theme === "dark" ? ColorScheme.DARK : ColorScheme.LIGHT,
  };
}
