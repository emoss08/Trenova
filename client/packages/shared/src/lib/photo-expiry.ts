/**
 * Signed picture links expire. A picture that fails to load is reported here once per link, and
 * whatever registered as a refresher (`registerPhotoRefresh`, called once by each app at start)
 * reads the data that carried the link again, so the next render draws a fresh link. Callers of
 * the avatar wire nothing: `AvatarImage` reports its own failures.
 *
 * This module is kept free of imports so the avatar, which every page draws, carries nothing with
 * it; the refreshers live in `photo-refresh.ts`.
 */

export type PhotoRefresher = (link: string) => void;

const refreshers = new Set<PhotoRefresher>();

/** Links already reported, so a picture that keeps failing asks for fresh data only once. */
const reported = new Set<string>();
const REPORTED_LIMIT = 500;

/** Registers what reads the data behind an expired link again; returns its removal. */
export function onExpiredPhoto(refresher: PhotoRefresher): () => void {
  refreshers.add(refresher);
  return () => {
    refreshers.delete(refresher);
  };
}

/** Says a picture link failed to load. Each link is passed to the refreshers at most once. */
export function reportExpiredPhoto(link: string): void {
  if (!link || reported.has(link)) return;
  if (reported.size >= REPORTED_LIMIT) {
    reported.clear();
  }
  reported.add(link);
  for (const refresher of refreshers) {
    refresher(link);
  }
}

/** The link without its query string: the same picture signed again keeps the same path. */
export function photoPath(link: string): string {
  const query = link.indexOf("?");
  return query < 0 ? link : link.slice(0, query);
}

/** Forgets what was reported; for tests. */
export function resetExpiredPhotos(): void {
  reported.clear();
  refreshers.clear();
}
