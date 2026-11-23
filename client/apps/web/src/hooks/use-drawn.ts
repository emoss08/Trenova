import { useEffect, useState } from "react";

/**
 * Whether a piece that grows into place has had its first frame: false while it is first painted
 * (at zero), true from the next frame, so a CSS transition carries it to its value. It stays true
 * after, so coming back to a kept view does not replay it. `ready` holds the first frame back until
 * the piece can be drawn at all, such as until it has been measured.
 *
 *   const drawn = useDrawn();
 *   <span style={{ width: drawn ? `${share}%` : 0 }} className="transition-[width]" />
 */
export function useDrawn(ready = true): boolean {
  const [drawn, setDrawn] = useState(false);
  useEffect(() => {
    if (drawn || !ready) return undefined;
    const frame = requestAnimationFrame(() => setDrawn(true));
    return () => cancelAnimationFrame(frame);
  }, [drawn, ready]);
  return drawn;
}
