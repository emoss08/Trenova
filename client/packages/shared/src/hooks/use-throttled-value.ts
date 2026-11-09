import { useEffect, useRef, useState } from "react";

/**
 * A value that follows its input at most once every `interval` while the input keeps moving,
 * and always lands on the last one. Where `useSettledValue` waits for the input to stop, this
 * keeps answering during a drag: a slider's readout moves with the thumb, a few times a second.
 */
export function useThrottledValue<T>(value: T, interval: number): T {
  const [throttled, setThrottled] = useState(value);
  const lastEmit = useRef(Number.NEGATIVE_INFINITY);

  useEffect(() => {
    const wait = Math.max(0, lastEmit.current + interval - Date.now());
    const handle = setTimeout(() => {
      lastEmit.current = Date.now();
      setThrottled(value);
    }, wait);

    return () => clearTimeout(handle);
  }, [value, interval]);

  return throttled;
}
