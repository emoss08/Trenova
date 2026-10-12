import { replaceEqualDeep } from "@tanstack/react-query";

/**
 * Structural sharing by key rather than by position, for derived lists
 * that grow at the front as well as the back.
 *
 * React Query keeps a query's data the same objects where nothing changed,
 * comparing arrays position by position. A transcript reading an older page
 * puts it in front, every position shifts, and that comparison keeps nothing.
 * Matching by key instead keeps every item that is still equal to what it was,
 * so a memoized row drawn from it is skipped.
 */
export function shareItemsByKey<T>(
  previous: T[] | undefined,
  next: T[],
  keyOf: (item: T) => string,
): T[] {
  if (previous === undefined || previous === next) {
    return next;
  }
  const before = new Map<string, T>();
  for (const item of previous) {
    before.set(keyOf(item), item);
  }

  let unchanged = previous.length === next.length;
  const shared = next.map((item, index) => {
    const prior = before.get(keyOf(item));
    const kept = prior === undefined ? item : replaceEqualDeep(prior, item);
    if (kept !== previous[index]) {
      unchanged = false;
    }
    return kept;
  });

  return unchanged ? previous : shared;
}

/** The same for a map: each value kept when it is still equal to the one under its key. */
export function shareMapValues<K, V>(
  previous: ReadonlyMap<K, V> | undefined,
  next: ReadonlyMap<K, V>,
): ReadonlyMap<K, V> {
  if (previous === undefined || previous === next) {
    return next;
  }

  let unchanged = previous.size === next.size;
  const shared = new Map<K, V>();
  for (const [key, value] of next) {
    const prior = previous.get(key);
    const kept = prior === undefined ? value : replaceEqualDeep(prior, value);
    if (kept !== prior) {
      unchanged = false;
    }
    shared.set(key, kept);
  }

  return unchanged ? previous : shared;
}
