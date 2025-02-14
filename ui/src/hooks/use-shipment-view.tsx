import { useCallback } from "react";
import { type ShipmentView, useShipmentParams } from "./use-shipment-params";

const STORAGE_KEY = "Trenova-preferred-shipment-view";
export function useShipmentView() {
  const { view, updateParams, isTransitioning } = useShipmentParams();

  const setViewAndStore = useCallback(
    (newView: ShipmentView | ((prev: ShipmentView) => ShipmentView)) => {
      try {
        const resolvedView =
          typeof newView === "function"
            ? newView((view as ShipmentView) || "list")
            : newView;

        localStorage.setItem(STORAGE_KEY, resolvedView);
        updateParams({ view: resolvedView });
      } catch (err) {
        console.warn("failed to save view preference:", err);
        updateParams({ view: newView as ShipmentView });
      }
    },
    [view, updateParams],
  );

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
