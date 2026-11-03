import type { WeatherAlertFeature } from "@/types/weather-alert";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMapFocusStore } from "./map-focus-store";
import type { MapShipment } from "./map-shipment-model";
import { ShipmentFocusCard } from "./shipment-focus-card";
import { WeatherAlertDetailCard } from "./weather-alert-detail-panel";
import type { AlertMatches } from "./weather-alert-matching";

const NO_HITS: readonly never[] = [];

/**
 * The one detail slot on the map, shared by the selected shipment and the
 * selected alert. It sits on the right of a wide map and across the top of a
 * narrow one.
 */
export function ShipmentMapCard({
  modelsById,
  featuresById,
  matches,
}: {
  modelsById: ReadonlyMap<string, MapShipment>;
  featuresById: ReadonlyMap<string, WeatherAlertFeature>;
  matches: AlertMatches;
}) {
  const t = useT();
  const focus = useMapFocusStore((s) => s.focus);
  const clearFocus = useMapFocusStore((s) => s.clearFocus);

  if (!focus) return null;

  let body: React.ReactNode = null;
  let label = "";
  if (focus.kind === "shipment") {
    const model = modelsById.get(focus.id);
    if (!model) return null;
    label = t("Shipment details");
    body = (
      <ShipmentFocusCard
        model={model}
        hits={matches.byShipment.get(model.id) ?? NO_HITS}
        featuresById={featuresById}
        onClose={clearFocus}
      />
    );
  } else {
    const feature = featuresById.get(focus.id);
    if (!feature) return null;
    label = t("Weather alert details");
    body = (
      <WeatherAlertDetailCard
        key={feature.properties.id}
        feature={feature}
        hits={matches.byAlert.get(feature.properties.id) ?? NO_HITS}
        modelsById={modelsById}
        onClose={clearFocus}
      />
    );
  }

  // The whole card scrolls, header and all, once it is taller than the map
  // allows. The section places and caps the card: the scroll area's root sets
  // its own `position: relative`, so it cannot be the positioned element.
  return (
    <section
      aria-label={label}
      className="ui-lift bg-card absolute top-3 right-3 z-10 flex max-h-[calc(100%-1.5rem)] w-80 flex-col overflow-hidden rounded-lg @max-3xl/map:inset-x-3 @max-3xl/map:max-h-[55%] @max-3xl/map:w-auto"
    >
      <ScrollArea className="min-h-0" maskVariant="card">
        {body}
      </ScrollArea>
    </section>
  );
}
