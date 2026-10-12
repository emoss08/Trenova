"use no memo";
import { shareItemsByKey, shareMapValues } from "@/lib/keyed-sharing";
import { useLayoutEffect, useMemo, useRef } from "react";

/**
 * A derived list whose items stay the same objects across renders while they
 * are equal, matched by key (see shareItemsByKey). Compared against what was
 * last committed, so a render React throws away never becomes the baseline.
 * `keyOf` is read once per change of `next`, so it should not depend on render.
 */
export function useSharedItems<T>(next: T[], keyOf: (item: T) => string): T[] {
  const committed = useRef<T[] | undefined>(undefined);
  // oxlint-disable-next-line react-hooks/exhaustive-deps -- keyOf is a fixed accessor
  const shared = useMemo(() => shareItemsByKey(committed.current, next, keyOf), [next]);
  useLayoutEffect(() => {
    committed.current = shared;
  }, [shared]);

  return shared;
}

/** A derived map whose values stay the same objects across renders while they are equal. */
export function useSharedMap<K, V>(next: ReadonlyMap<K, V>): ReadonlyMap<K, V> {
  const committed = useRef<ReadonlyMap<K, V> | undefined>(undefined);
  const shared = useMemo(() => shareMapValues(committed.current, next), [next]);
  useLayoutEffect(() => {
    committed.current = shared;
  }, [shared]);

  return shared;
}
