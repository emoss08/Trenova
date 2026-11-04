import { useState } from "react";

/**
 * Whether something has ever been open. What loads on demand mounts the first time it opens
 * and then stays mounted, so closing it can still play out and reopening costs nothing.
 */
export function useOpenedOnce(open: boolean): boolean {
  const [opened, setOpened] = useState(open);
  if (open && !opened) setOpened(true);
  return opened;
}
