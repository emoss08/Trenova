import { useState } from "react";

/**
 * `value` as it was the last time `settled` held: while something it describes is still on its
 * way (a page of another size, a regrouped list), the caller can tell what is on screen from
 * what has been asked for. It follows `value` in the same render that `settled` turns true, so
 * nothing is drawn a frame behind. `same` says when a new value is the one kept (a list rebuilt
 * with the same items), so it is not taken again.
 */
export function useLastSettled<T>(
  value: T,
  settled: boolean,
  same: (a: T, b: T) => boolean = Object.is,
): T {
  const [kept, setKept] = useState(value);
  if (settled && !same(kept, value)) {
    setKept(value);
    return value;
  }
  return kept;
}
