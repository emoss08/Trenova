import { useEffect, useState } from "react";

/**
 * Keeps an item open: whenever none is, the first one the queue lists is opened, so
 * the page always lands on work. `firstItemId` is what the list last reported:
 * undefined until it has answered, null when it lists nothing.
 */
export function useOpenFirstItem(selectedItemId: string | null, open: (itemId: string) => void) {
  const [firstItemId, setFirstItemId] = useState<string | null | undefined>(undefined);

  useEffect(() => {
    if (!selectedItemId && firstItemId) {
      open(firstItemId);
    }
  }, [selectedItemId, firstItemId, open]);

  return { firstItemId, reportFirstItem: setFirstItemId };
}
