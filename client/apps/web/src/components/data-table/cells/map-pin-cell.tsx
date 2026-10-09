import { LinkExternal01Icon, MarkerPin01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy, Suspense, useState } from "react";

const LazyMapWithKey = lazy(() =>
  import("@/components/lazy-map").then((module) => ({ default: module.LazyMapWithKey })),
);

type MapPinCellProps = {
  latitude: number | null | undefined;
  longitude: number | null | undefined;
  /** What the pin is, as a person reads it: a city, an address, a facility. */
  label: string;
  /** The full address shown above the map. */
  address?: string;
};

function hasCoordinates(latitude: unknown, longitude: unknown): boolean {
  return (
    typeof latitude === "number" &&
    typeof longitude === "number" &&
    Number.isFinite(latitude) &&
    Number.isFinite(longitude) &&
    Math.abs(latitude) <= 90 &&
    Math.abs(longitude) <= 180 &&
    !(latitude === 0 && longitude === 0)
  );
}

/**
 * A place with a pin that opens it on a map. The map, its script and its key are
 * loaded only when the pin is opened, so a page of pins costs nothing until then.
 */
export function MapPinCell({ latitude, longitude, label, address }: MapPinCellProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  if (!hasCoordinates(latitude, longitude)) {
    return <span className="truncate text-sm">{label || "—"}</span>;
  }
  const position = { lat: latitude as number, lng: longitude as number };
  const externalHref = `https://www.google.com/maps/search/?api=1&query=${position.lat},${position.lng}`;

  return (
    <span className="flex min-w-0 items-center gap-1.5">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Show {0} on a map", label)}
              onClick={(event) => event.stopPropagation()}
            >
              <MarkerPin01Icon className="size-3.5" />
            </Button>
          }
        />
        <PopoverContent align="start" className="w-80 p-0" onClick={(event) => event.stopPropagation()}>
          <div className="flex items-start justify-between gap-2 px-3 py-2">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{label}</p>
              {address ? <p className="text-muted-foreground truncate text-xs">{address}</p> : null}
            </div>
            <a
              href={externalHref}
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground ui-focus-ring shrink-0 rounded-sm"
              aria-label={t("Open in Google Maps")}
            >
              <LinkExternal01Icon className="size-3.5" />
            </a>
          </div>
          <div className="h-48 overflow-hidden rounded-b-lg">
            {open ? (
              <Suspense fallback={<Skeleton className="size-full rounded-none" />}>
                <LazyMapWithKey position={position} />
              </Suspense>
            ) : null}
          </div>
        </PopoverContent>
      </Popover>
      <span className="truncate text-sm">{label}</span>
    </span>
  );
}
