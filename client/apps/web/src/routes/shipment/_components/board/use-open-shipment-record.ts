import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { useCallback } from "react";
import { useShipmentBoardUrl } from "./url-state";

/**
 * Opens a shipment on the board the dispatcher is already looking at: the row
 * expands and the editor opens, the same state a record link
 * (config/record-links.ts) lands on. Only the URL changes, so the board's
 * view, grouping and filters stay and nothing reloads.
 */
export function useOpenShipmentRecord() {
  const [, setBoardUrl] = useShipmentBoardUrl();
  const [, setPanel] = useSearchState(panelSearchParamsParser);

  return useCallback(
    (shipmentId: string) => {
      void setBoardUrl({ expanded: shipmentId });
      void setPanel({ panelType: "edit", panelEntityId: shipmentId });
    },
    [setBoardUrl, setPanel],
  );
}
