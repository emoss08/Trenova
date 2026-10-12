import type { PresenceTarget } from "@/hooks/use-presence-viewers";

const BASE = "/realtime/presence";

/**
 * Who is looking at a table. The server builds the same scope from the resource in
 * the path, after checking the caller may read it.
 */
export function viewPresence(resource: string): PresenceTarget {
  return {
    scope: `view:${resource}`,
    joinPath: `${BASE}/${encodeURIComponent(resource)}/`,
  };
}

/** Who has one record open. */
export function recordPresence(resource: string, recordId: string): PresenceTarget {
  return {
    scope: `record:${resource}:${recordId}`,
    joinPath: `${BASE}/${encodeURIComponent(resource)}/${encodeURIComponent(recordId)}/`,
  };
}
