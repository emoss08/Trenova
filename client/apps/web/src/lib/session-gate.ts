import { clearSessionCache } from "@/lib/session-cache";
import { requireSession } from "@trenova/shared/lib/session-loader";

/**
 * The web app's session gate. A session the server refused ended while nothing in this tab
 * saw it end (it expired overnight, or another tab signed out), so what the browser kept for
 * it is cleared before the sign-in page shows, as for any other session end.
 */
export async function requireWebSession(loginPath: string): Promise<Response | null> {
  const sessionRedirect = await requireSession(loginPath);
  if (sessionRedirect) {
    await clearSessionCache();
  }
  return sessionRedirect;
}
