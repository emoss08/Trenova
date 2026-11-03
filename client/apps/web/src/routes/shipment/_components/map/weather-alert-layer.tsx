import type { WeatherAlertFeature } from "@/types/weather-alert";
import { useMapFocusStore } from "./map-focus-store";
import { WeatherAlertPolygon } from "./weather-alert-polygon";

/**
 * Public weather alert areas in their category colours. Clicking one opens it
 * in the map's detail card and the rail's alerts tab; while one is open the
 * others step back.
 */
export function WeatherAlertLayer({ features }: { features: readonly WeatherAlertFeature[] }) {
  const focus = useMapFocusStore((s) => s.focus);
  const hoverId = useMapFocusStore((s) => s.alertHoverId);
  const toggleAlert = useMapFocusStore((s) => s.toggleAlert);
  const setRailTab = useMapFocusStore((s) => s.setRailTab);
  const setAlertHoverId = useMapFocusStore((s) => s.setAlertHoverId);
  const selectedId = focus?.kind === "alert" ? focus.id : null;

  return (
    <>
      {features.map((feature) => {
        const id = feature.properties.id;
        const state =
          id === selectedId
            ? "selected"
            : id === hoverId
              ? "hovered"
              : selectedId
                ? "dimmed"
                : "idle";
        return (
          <WeatherAlertPolygon
            key={id}
            feature={feature}
            state={state}
            onClick={() => {
              toggleAlert(id);
              setRailTab("alerts");
            }}
            onHoverChange={(hovered) => {
              if (hovered) setAlertHoverId(id);
              else if (useMapFocusStore.getState().alertHoverId === id) setAlertHoverId(null);
            }}
          />
        );
      })}
    </>
  );
}
