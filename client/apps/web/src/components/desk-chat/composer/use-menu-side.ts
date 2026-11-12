import { useCallback, useState } from "react";

/** The room a composer menu wants above the box before it opens there. */
const MENU_ROOM = 440;

/**
 * Which side of the composer a menu opens on: above it, as in a conversation
 * where the box sits at the foot of the page, unless there is not the room
 * and more of it below, as on the front page where the box sits near the top.
 * Measured once as the menu mounts, so it never jumps sides while open.
 */
export function useMenuSide() {
  const [below, setBelow] = useState(false);
  const attach = useCallback((menu: HTMLElement | null) => {
    const frame = menu?.closest(".dk-cmp");
    if (!frame) {
      return;
    }
    const rect = frame.getBoundingClientRect();
    const above = rect.top;
    const under = window.innerHeight - rect.bottom;
    setBelow(above < MENU_ROOM && under > above);
  }, []);

  return { attach, below };
}
