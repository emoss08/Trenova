import { useLocalStorage } from "@/hooks/use-local-storage";
import { useCallback } from "react";
import { useShipmentBoardUrl } from "./url-state";

const PANEL_OPEN_STORAGE_KEY = "trenova.shipments.panel-open";

export function useBoardPanel(): [boolean, (open: boolean) => void] {
  const [{ panel }, setUrl] = useShipmentBoardUrl();
  const [rememberedOpen, setRememberedOpen] = useLocalStorage(PANEL_OPEN_STORAGE_KEY, false);
  const setPanelOpen = useCallback(
    (open: boolean) => {
      setRememberedOpen(open);
      void setUrl({ panel: open });
    },
    [setRememberedOpen, setUrl],
  );
  return [panel ?? rememberedOpen, setPanelOpen];
}
