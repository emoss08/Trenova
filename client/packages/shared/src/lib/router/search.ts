/**
 * The query string as every router in the client reads and writes it: plain strings, one
 * value per key or a list for a repeated key, the way the server, nuqs and the links in
 * emails write them. The router's default would read `?id=00123` as the number 123 and
 * write a string that looks like a number back quoted, changing addresses people saved.
 */
export type SearchRecord = Record<string, string | string[]>;

// The characters a value cannot carry raw, and nothing more, so an address stays as
// readable as nuqs writes it (commas, colons and slashes are left alone).
function encodePart(input: string): string {
  return input
    .replace(/%/g, "%25")
    .replace(/\+/g, "%2B")
    .replace(/ /g, "+")
    .replace(/#/g, "%23")
    .replace(/&/g, "%26")
    .replace(/=/g, "%3D")
    .replace(/\?/g, "%3F")
    .replace(/"/g, "%22")
    .replace(/'/g, "%27")
    .replace(/`/g, "%60")
    .replace(/</g, "%3C")
    .replace(/>/g, "%3E")
    .split("")
    .map((char) => (char.charCodeAt(0) < 0x20 ? encodeURIComponent(char) : char))
    .join("");
}

function stringValue(value: unknown): string {
  return typeof value === "object" ? JSON.stringify(value) : String(value);
}

export function parseSearch(searchStr: string): SearchRecord {
  const params = new URLSearchParams(searchStr.startsWith("?") ? searchStr.slice(1) : searchStr);
  const search: SearchRecord = {};
  for (const key of new Set(params.keys())) {
    const values = params.getAll(key);
    search[key] = values.length > 1 ? values : (values[0] ?? "");
  }
  return search;
}

export function stringifySearch(search: Record<string, unknown>): string {
  const parts: string[] = [];
  for (const [key, value] of Object.entries(search)) {
    if (value === undefined || value === null) continue;
    for (const item of Array.isArray(value) ? value : [value]) {
      if (item === undefined || item === null) continue;
      parts.push(`${encodePart(key)}=${encodePart(stringValue(item))}`);
    }
  }
  return parts.length > 0 ? `?${parts.join("&")}` : "";
}

/**
 * The root route's `validateSearch`: every route's query is a SearchRecord, so a link may name
 * any key and a page reads strings, never numbers or booleans the router guessed at.
 */
export function validateSearchRecord(search: Record<string, unknown>): SearchRecord {
  return search as SearchRecord;
}
