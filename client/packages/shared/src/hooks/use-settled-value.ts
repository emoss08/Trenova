import { useEffect, useRef, useState } from "react";

/**
 * A value that follows its input at once after a quiet spell, but passes
 * over changes that come faster than `delay` apart and lands on the last.
 * Where `useDebounce` always waits, this waits only while the input is
 * moving: a click is answered straight away, a held key only where it stops.
 */
export function useSettledValue<T>(value: T, delay: number): T {
  const [settled, setSettled] = useState(value);
  const lastChange = useRef(Number.NEGATIVE_INFINITY);

  useEffect(() => {
    const now = Date.now();
    const quiet = now - lastChange.current >= delay;
    lastChange.current = now;
    const handle = setTimeout(() => setSettled(value), quiet ? 0 : delay);

    return () => clearTimeout(handle);
  }, [value, delay]);

  return settled;
}
