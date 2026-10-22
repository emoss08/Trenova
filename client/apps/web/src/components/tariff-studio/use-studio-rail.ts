import { useLocalStorage } from "@/hooks/use-local-storage";
import { useCallback, useState } from "react";

export const RAIL_OPEN_STORAGE_KEY = "trenova.formula-studio.rail-open";

/**
 * Whether a Studio's floating panel is open (templates and libraries share the choice). Beside the form it stays as the
 * person last left it, from one visit to the next. Over the form, on a narrow
 * screen, it starts closed every time, and opening or closing it there leaves
 * the wide screen's choice alone.
 */
export function useStudioRail(stacked: boolean): [boolean, (open: boolean) => void] {
  const [remembered, setRemembered] = useLocalStorage(RAIL_OPEN_STORAGE_KEY, true);
  const [stackedOpen, setStackedOpen] = useState(false);
  const setOpen = useCallback(
    (open: boolean) => {
      if (stacked) {
        setStackedOpen(open);
        return;
      }
      setRemembered(open);
    },
    [stacked, setRemembered],
  );
  return [stacked ? stackedOpen : remembered, setOpen];
}
