import { parseAsString, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useEffect } from "react";

const panelParsers = {
  panelType: parseAsStringLiteral(["edit", "create"] as const),
  panelEntityId: parseAsString,
};

export type AddressedPanel =
  | { mode: "edit"; entityId: string }
  | { mode: "create"; preset: string | null };

/**
 * Opens an editor another tab asked for in the address, once the tab can, and
 * clears the request so going back does not open it again. `open` returns
 * false while the record it names has not loaded yet.
 */
export function useAddressedPanel(open: (panel: AddressedPanel) => boolean) {
  const [{ panelType, panelEntityId }, setPanel] = useQueryStates(panelParsers);

  useEffect(() => {
    if (!panelType) {
      return;
    }
    const request: AddressedPanel | null =
      panelType === "create"
        ? { mode: "create", preset: panelEntityId }
        : panelEntityId
          ? { mode: "edit", entityId: panelEntityId }
          : null;
    if (request && !open(request)) {
      return;
    }
    void setPanel({ panelType: null, panelEntityId: null }, { history: "replace" });
  }, [open, panelEntityId, panelType, setPanel]);
}
