/**
 * The values a form keeps editing after a save that leaves it open: what it
 * holds, with the version the save gave the record.
 *
 * Every save bumps the record's version and refuses a stale one, so a form
 * that keeps the version it was opened with has its next save refused as out
 * of date. A save that returns no version, or a form that holds none, keeps
 * the values as they are.
 */
export function withSavedVersion<TValues extends object>(values: TValues, saved: unknown): TValues {
  if (!("version" in values) || saved === null || typeof saved !== "object") {
    return values;
  }
  const version = (saved as { version?: unknown }).version;
  if (typeof version !== "number") {
    return values;
  }
  return { ...values, version };
}
