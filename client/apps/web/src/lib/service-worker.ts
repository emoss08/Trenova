import { reloadIsSafe } from "@/lib/reload-guards";
import type { DataRouter, Location as RouterLocation } from "react-router";

// How often an open tab asks whether a new release is out. The browser checks on its own only
// when a page loads, which a single-page app rarely does, so a tab left open all day would
// otherwise run the release it was opened on. Coming back to the tab asks too, but never
// more than once a minute however often the tab is switched to.
const RELEASE_CHECK_INTERVAL_MS = 30 * 60 * 1000;
const RELEASE_CHECK_MIN_GAP_MS = 60 * 1000;

// How long a reload that is recovering from a release waits for the release's worker to take
// over before reloading anyway.
const ACTIVATION_WAIT_MS = 5000;

/**
 * Registers the web app's service worker (src/service-worker/sw.ts) once the page has finished
 * loading, so it never competes with the app's own start; asks for new releases while the tab
 * stays open; and moves the page onto a new release at its next change of page. Production
 * builds only: on the dev server the code changes under any cache a worker kept. A build made
 * with VITE_SERVICE_WORKER=off registers nothing, and the sw.js it ships removes the worker
 * from browsers that installed one before.
 */
export function registerServiceWorker(router: DataRouter): void {
  if (!serviceWorkerEnabled()) return;

  reloadIntoNewRelease({ router, container: navigator.serviceWorker, reloadIsSafe });

  const base = import.meta.env.BASE_URL;
  const register = () => {
    navigator.serviceWorker
      .register(`${base}sw.js`, { scope: base, updateViaCache: "none" })
      .then((registration) => {
        const check = releaseCheck(registration);
        window.setInterval(check, RELEASE_CHECK_INTERVAL_MS);
        document.addEventListener("visibilitychange", () => {
          if (document.visibilityState === "visible") check();
        });
      })
      .catch((error: unknown) => console.error("Could not register the service worker:", error));
  };

  if (document.readyState === "complete") register();
  else window.addEventListener("load", register, { once: true });
}

function serviceWorkerEnabled(): boolean {
  return (
    import.meta.env.PROD &&
    import.meta.env.VITE_SERVICE_WORKER !== "off" &&
    typeof navigator !== "undefined" &&
    "serviceWorker" in navigator
  );
}

/**
 * A check for a new release that runs at most once a minute. A check that fails (offline, the
 * server restarting) is simply made again at the next one.
 */
export function releaseCheck(
  registration: Pick<ServiceWorkerRegistration, "update">,
  now: () => number = Date.now,
): () => void {
  let lastCheck = Number.NEGATIVE_INFINITY;
  return () => {
    if (now() - lastCheck < RELEASE_CHECK_MIN_GAP_MS) return;
    lastCheck = now();
    void registration.update().catch(() => undefined);
  };
}

/**
 * Asks whether a newer release is out and, when it is, waits (a few seconds at most) for its
 * worker to take over, so that a reload made next opens the new release rather than the
 * cached old one. Resolves at once without a worker; never rejects.
 */
export async function settleOnNewestRelease(
  container: Pick<ServiceWorkerContainer, "getRegistration"> | undefined = serviceWorkerEnabled()
    ? navigator.serviceWorker
    : undefined,
  waitMs: number = ACTIVATION_WAIT_MS,
): Promise<void> {
  try {
    const registration = await container?.getRegistration();
    if (!registration) return;
    await registration.update();
    const incoming = registration.installing ?? registration.waiting;
    if (!incoming) return;
    await new Promise<void>((resolve) => {
      const timer = setTimeout(resolve, waitMs);
      incoming.addEventListener("statechange", () => {
        if (incoming.state === "activated" || incoming.state === "redundant") {
          clearTimeout(timer);
          resolve();
        }
      });
    });
  } catch {
    // Offline, or the worker cannot be reached: the reload goes ahead with what there is.
  }
}

type ReleaseRouter = Pick<DataRouter, "state" | "subscribe" | "createHref">;
type WorkerContainer = Pick<ServiceWorkerContainer, "controller" | "addEventListener">;
type PageNavigator = { location: Pick<Location, "assign"> };

/**
 * Once a newer release's worker has taken over this page, the next change of page is a full
 * page load of where the person was going, which opens the new release.
 *
 * - A change of the query string alone (a filter, a sort, a table page) is not a change of
 *   page and never reloads.
 * - A page that had no worker when it loaded is already the newest build, so the worker's
 *   first install is not a release.
 * - While `reloadIsSafe` says no — an unsaved form, an open popout — the page changes in place
 *   as before, and the reload waits for a later change of page made while nothing holds it.
 * - If the browser stays on the page after all (someone answered "Stay" to a prompt), the next
 *   change of page tries again.
 *
 * Returns the router unsubscribe, for tests.
 */
export function reloadIntoNewRelease({
  router,
  container,
  reloadIsSafe: safe,
  target = window,
}: {
  router: ReleaseRouter;
  container: WorkerContainer;
  reloadIsSafe: () => boolean;
  target?: PageNavigator;
}): () => void {
  let controlled = container.controller !== null;
  let releaseReady = false;
  container.addEventListener("controllerchange", () => {
    if (controlled) releaseReady = true;
    controlled = true;
  });

  let committedPath = router.state.location.pathname;
  let leavingFor: string | null = null;
  const leaveFor = (location: RouterLocation) => {
    const href = router.createHref(location);
    if (href === leavingFor) return;
    leavingFor = href;
    target.location.assign(href);
  };

  return router.subscribe((state) => {
    const next = state.navigation.location;
    if (next) {
      if (releaseReady && next.pathname !== committedPath && safe()) leaveFor(next);
      return;
    }
    if (state.location.pathname === committedPath) return;
    committedPath = state.location.pathname;
    if (releaseReady && safe()) leaveFor(state.location);
  });
}
