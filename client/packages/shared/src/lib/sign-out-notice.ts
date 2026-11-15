import { RETURN_PATH_PARAM } from "@trenova/shared/lib/router/return-path";

const SIGNED_OUT_PARAM = "signed_out";

/**
 * Why the sign-in page is being shown, when the person did not sign out in this tab and
 * should be told: the idle timeout ran out, the server had already ended the session, or
 * they signed out in another tab.
 */
export type SignOutNotice = "idle" | "expired" | "elsewhere";

const NOTICES: ReadonlySet<string> = new Set<SignOutNotice>(["idle", "expired", "elsewhere"]);

/**
 * The sign-in page to land on once a session has ended or was never there, carrying the notice
 * if there is one and the address to return to once signed in. `returnTo` must come from
 * `returnPathFor`, which leaves out home and anything unsafe.
 */
export function signInPathAfter(
  loginPath: string,
  notice?: SignOutNotice,
  returnTo?: string | null,
): string {
  const search = new URLSearchParams();
  if (notice) {
    search.set(SIGNED_OUT_PARAM, notice);
  }
  if (returnTo) {
    search.set(RETURN_PATH_PARAM, returnTo);
  }

  const query = search.toString();
  return query ? `${loginPath}?${query}` : loginPath;
}

/** The notice the sign-in page was opened with, ignoring anything it does not know. */
export function signOutNoticeFrom(search: URLSearchParams): SignOutNotice | null {
  const value = search.get(SIGNED_OUT_PARAM);
  return value !== null && NOTICES.has(value) ? (value as SignOutNotice) : null;
}

/** The search without the notice, once the person has read it or started signing in. */
export function withoutSignOutNotice(search: URLSearchParams): URLSearchParams {
  const next = new URLSearchParams(search);
  next.delete(SIGNED_OUT_PARAM);
  return next;
}
