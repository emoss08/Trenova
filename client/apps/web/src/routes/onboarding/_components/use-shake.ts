import { useCallback, useEffect, useRef, useState } from "react";

/**
 * A refusal shake that plays again on every refused attempt: the flag drops for one
 * frame and rises again, which restarts the CSS animation without remounting the
 * element (and so without losing focus).
 */
export function useShake(): [boolean, () => void] {
  const [shaking, setShaking] = useState(false);
  const frame = useRef<number | null>(null);

  useEffect(
    () => () => {
      if (frame.current !== null) {
        cancelAnimationFrame(frame.current);
      }
    },
    [],
  );

  const shake = useCallback(() => {
    setShaking(false);
    if (frame.current !== null) {
      cancelAnimationFrame(frame.current);
    }
    frame.current = requestAnimationFrame(() => {
      frame.current = null;
      setShaking(true);
    });
  }, []);

  return [shaking, shake];
}
