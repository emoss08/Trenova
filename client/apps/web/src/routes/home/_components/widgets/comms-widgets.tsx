import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { lazy, Suspense } from "react";
import { WidgetEmpty, WidgetShell } from "../widget-shell";
import type { WidgetProps } from "../widget-registry";

/**
 * The live fleet map is the heaviest thing on the canvas — Google Maps, vehicle
 * positions, geofences, and weather layers — so it loads only once a layout
 * actually places it, and it is the same panel the dispatch board draws rather
 * than a second, thinner map that would drift from it.
 */
const ShipmentMapPanel = lazy(() => import("@/routes/shipment/_components/map/shipment-map-panel"));

/**
 * An administrator's note to everyone assigned this home screen. It renders as
 * plain text with line breaks preserved rather than as markup: an announcement
 * is authored in a textarea by someone who is not writing HTML, and rendering
 * unescaped author input on every user's first screen is not a risk worth
 * taking for italics.
 */
export function AnnouncementWidget({ widget }: WidgetProps) {
  const text = widget.config.text?.trim();

  return (
    <WidgetShell title={widget.title || "Announcement"}>
      {!text ? (
        <WidgetEmpty>Nothing announced.</WidgetEmpty>
      ) : (
        <p className="text-foreground/90 text-xs leading-relaxed whitespace-pre-wrap">{text}</p>
      )}
    </WidgetShell>
  );
}

export function MapWidget({ widget }: WidgetProps) {
  return (
    <WidgetShell
      title={widget.title || "Fleet Map"}
      href="/shipment-management/shipments"
      hrefLabel="Open dispatch board"
      scroll={false}
      bodyClassName="p-0"
    >
      <div className="min-h-0 flex-1 overflow-hidden">
        <Suspense fallback={<Skeleton className="size-full rounded-none" />}>
          <ShipmentMapPanel />
        </Suspense>
      </div>
    </WidgetShell>
  );
}
