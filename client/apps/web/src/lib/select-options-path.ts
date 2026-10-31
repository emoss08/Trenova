/**
 * A REST select-options request, as `api.get` takes it. `ids` asks for those records instead
 * of a search: a REST select-options endpoint that a multi-select names answers `ids` in one
 * read (`/documents/select-options/`). An array in `extraSearchParams` is sent once per entry.
 */
export function selectOptionsPath(
  link: string,
  params: Record<string, string>,
  extraSearchParams?: Record<string, string | string[] | undefined>,
  ids: readonly string[] = [],
): string {
  const search = new URLSearchParams(params);
  for (const [key, value] of Object.entries(extraSearchParams ?? {})) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) {
      for (const entry of value) search.append(key, entry);
    } else {
      search.set(key, value.toString());
    }
  }
  for (const id of ids) {
    search.append("ids", id);
  }
  const query = search.toString();
  return query ? `${link}?${query}` : link;
}
