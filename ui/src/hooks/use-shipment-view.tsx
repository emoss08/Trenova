import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useEffect, useTransition } from "react";

export const SHIPMENT_VIEWS = ["list", "map"] as const;
export type ShipmentView = (typeof SHIPMENT_VIEWS)[number];

const STORAGE_KEY = "Trenova-preferred-shipment-view";
const viewParams = parseAsStringLiteral(SHIPMENT_VIEWS);

export function getStoredView() {
  try {
    const storedView = localStorage.getItem(STORAGE_KEY);
    console.log("Stored view from localStorage:", storedView); // Debug log

    return storedView && SHIPMENT_VIEWS.includes(storedView as ShipmentView)
      ? (storedView as ShipmentView)
      : "list";
  } catch {
    console.warn("failed to access localstorage, defaulting to list view");
    return "list";
  }
}

export function useShipmentView() {
  const [isTransitioning, startTransition] = useTransition();

  const [view, setView] = useQueryState(
    "view",
    viewParams.withOptions({
      startTransition,
      shallow: false,
    }),
  );

  // Initialize from localstorage if the URL param is not set
  useEffect(() => {
    console.log("Current view:", view); // Debug log
    if (!view) {
      const storedView = getStoredView();
      console.log("Setting view from storage:", storedView); // Debug log
      setView(storedView);
    }
  }, [view, setView]);

  // Wrapper for setting the view and storing it in localstorage
  const setViewAndStore = useCallback(
    (newView: ShipmentView | ((prev: ShipmentView) => ShipmentView)) => {
      try {
        const resolvedView =
          typeof newView === "function" ? newView(view || "list") : newView;

        console.log("Saving new view to localStorage:", resolvedView); // Debug log
        localStorage.setItem(STORAGE_KEY, resolvedView);
        setView(resolvedView);
      } catch (err) {
        console.warn("failed to save view preference:", err);
        setView(newView as ShipmentView);
      }
    },
    [view, setView],
  );

  console.log("Current view in hook:", view); // Debug log

  return {
    view: view as ShipmentView,
    setView: setViewAndStore,
    isTransitioning,
    clearViewPreference: () => {
      try {
        localStorage.removeItem(STORAGE_KEY);
      } catch (err) {
        console.warn("failed to clear view preference:", err);
      }
    },
  };
}
