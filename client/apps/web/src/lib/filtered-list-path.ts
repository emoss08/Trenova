import type { FieldFilter } from "@trenova/shared/types/data-table";

/**
 * A link to a data table page with filters already applied, in the `fieldFilters` form the
 * table reads from its URL, so a screen can open a list narrowed to what it was showing.
 */
export function filteredListPath(path: string, filters: readonly FieldFilter[]): string {
  if (filters.length === 0) return path;
  const params = new URLSearchParams({ fieldFilters: JSON.stringify(filters) });
  return `${path}?${params.toString()}`;
}
