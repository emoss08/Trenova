import type { Query, QueryClient } from "@tanstack/react-query";
import { userService } from "@trenova/shared/services/user";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { onExpiredPhoto } from "./photo-expiry";

/** How deep a cached read is searched for a link; a photo sits a few levels into a record. */
const MAX_DEPTH = 12;

/** Whether a value read from the server carries the link anywhere inside it. */
export function holdsLink(value: unknown, link: string, depth = 0): boolean {
  if (value === link) return true;
  if (depth >= MAX_DEPTH || value === null || typeof value !== "object") return false;
  if (Array.isArray(value)) {
    return value.some((item) => holdsLink(item, link, depth + 1));
  }
  for (const item of Object.values(value)) {
    if (holdsLink(item, link, depth + 1)) return true;
  }
  return false;
}

function queriesHolding(client: QueryClient, link: string): Query[] {
  return client
    .getQueryCache()
    .findAll({ type: "active" })
    .filter((query) => holdsLink(query.state.data, link));
}

/**
 * Reads again whatever carried an expired picture link: every query on screen whose data holds
 * it, and the signed-in person's own record when the link is their photo. Each app calls this
 * once at start with its query client; it returns the removal.
 */
export function registerPhotoRefresh(client: QueryClient): () => void {
  return onExpiredPhoto((link) => {
    for (const query of queriesHolding(client, link)) {
      void client.invalidateQueries({ queryKey: query.queryKey, exact: true });
    }
    const me = useAuthStore.getState().user;
    if (me?.photo && (me.photo.url === link || me.photo.thumbnailUrl === link)) {
      void userService
        .currentUser()
        .then((fresh) => useAuthStore.getState().setUser(fresh))
        .catch(() => undefined);
    }
  });
}
