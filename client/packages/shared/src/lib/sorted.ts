/**
 * Where `value` would go in a list sorted by `keyOf`: before the first item
 * whose key is at or above it, or, with `after`, before the first whose key is
 * above it.
 */
function bisect<T>(
  sorted: readonly T[],
  value: number,
  keyOf: (item: T) => number,
  after: boolean,
): number {
  let low = 0;
  let high = sorted.length;
  while (low < high) {
    const middle = (low + high) >>> 1;
    const key = keyOf(sorted[middle]);
    if (key < value || (after && key === value)) low = middle + 1;
    else high = middle;
  }
  return low;
}

/**
 * The items of a list sorted by `keyOf` whose key lies between `from` and `to`,
 * both included, found by bisecting rather than by reading the whole list.
 */
export function sortedRange<T>(
  sorted: readonly T[],
  from: number,
  to: number,
  keyOf: (item: T) => number,
): T[] {
  if (to < from) return [];
  return sorted.slice(bisect(sorted, from, keyOf, false), bisect(sorted, to, keyOf, true));
}

/** The last item of a list sorted by `keyOf` whose key is at or before `value`. */
export function lastAtOrBefore<T>(
  sorted: readonly T[],
  value: number,
  keyOf: (item: T) => number,
): T | undefined {
  const index = bisect(sorted, value, keyOf, true);
  return index > 0 ? sorted[index - 1] : undefined;
}
