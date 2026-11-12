import {
  AdvancedMarker as LibraryAdvancedMarker,
  useMap,
  useMapsLibrary,
  type AdvancedMarkerProps,
} from "@vis.gl/react-google-maps";
import { forwardRef, useEffect, useMemo, useState } from "react";

let guarded = false;

/** The accessor a property is defined by, wherever on the prototype chain it lives. */
function accessorOf(
  target: object,
  name: string,
): { owner: object; own: PropertyDescriptor } | null {
  for (let owner: object | null = target; owner; owner = Object.getPrototypeOf(owner)) {
    const own = Object.getOwnPropertyDescriptor(owner, name);
    if (own) {
      return { owner, own };
    }
  }

  return null;
}

/**
 * Keeps taking a marker off its map from throwing. When a map is torn down
 * (the page changes, the panel closes, Google could not draw the map at all),
 * every marker is detached with `marker.map = null`, and Google's own setter
 * throws "Cannot read properties of undefined (reading 'getRootNode')" when
 * the map's element is already gone. Nothing is left to detach from by then,
 * so that one failure is dropped; putting a marker on a map still fails
 * loudly. Installed once, on the marker class itself, so every marker on the
 * site is covered whichever map drew it.
 */
function guardMarkerDetach(library: google.maps.MarkerLibrary): void {
  if (guarded) {
    return;
  }
  const found = accessorOf(library.AdvancedMarkerElement.prototype, "map");
  const assign = found?.own.set;
  if (!found || !assign) {
    return;
  }
  guarded = true;
  Object.defineProperty(found.owner, "map", {
    ...found.own,
    set(this: unknown, value: google.maps.Map | null | undefined) {
      try {
        assign.call(this, value);
      } catch (error) {
        if (value) {
          throw error;
        }
      }
    },
  });
}

/**
 * Whether Google can make a marker at all right now: null while its library
 * is still being read. When a key is refused or out of quota Google leaves
 * the map half built, and from then on constructing any marker throws inside
 * its own code ("reading 'keys'"). Making one that is put on no map shows
 * that up front, so a map can draw without markers, or fall back to
 * something else, instead of an error taking its place.
 */
export function useMarkersUsable(): boolean | null {
  const library = useMapsLibrary("marker");

  return useMemo(() => {
    if (!library) {
      return null;
    }
    try {
      guardMarkerDetach(library);
      // oxlint-disable-next-line no-new -- made only to learn whether making one throws
      new library.AdvancedMarkerElement();

      return true;
    } catch {
      return false;
    }
  }, [library]);
}

/** The maps that have drawn their tiles at least once. */
const drawn = new WeakSet<google.maps.Map>();

/**
 * Whether the map this is drawn in has painted its tiles at least once. A
 * map Google will not serve (a key it refuses, a key out of quota) never
 * does, and it is left half built: a marker put on it throws from inside
 * Google's own drawing loop on every frame, where nothing can catch it. So
 * nothing is put on a map until it has shown it can draw.
 */
export function useMapDrawn(): boolean {
  const map = useMap();
  const [ready, setReady] = useState(() => (map ? drawn.has(map) : false));

  useEffect(() => {
    if (!map) {
      setReady(false);
      return;
    }
    if (drawn.has(map)) {
      setReady(true);
      return;
    }
    setReady(false);
    const listener = google.maps.event.addListenerOnce(map, "tilesloaded", () => {
      drawn.add(map);
      setReady(true);
    });

    return () => listener.remove();
  }, [map]);

  return ready;
}

/**
 * The map marker every map on the site draws: the library's, with detaching
 * made safe, and left out until its map has drawn and while Google cannot
 * make markers.
 * Import this one, never the library's own.
 */
export const AdvancedMarker = forwardRef<
  google.maps.marker.AdvancedMarkerElement,
  AdvancedMarkerProps
>(function AdvancedMarker(props, ref) {
  const usable = useMarkersUsable();
  const mapDrawn = useMapDrawn();
  if (usable !== true || !mapDrawn) {
    return null;
  }

  return <LibraryAdvancedMarker ref={ref} {...props} />;
});
