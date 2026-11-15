import { edition } from "@/lib/edition";
import {
  loadOnboardingSnapshot,
  onboardingExitRedirect,
  resolveOnboardingRedirect,
} from "@/lib/onboarding-gate";
import { webSignInPaths } from "@/lib/sign-in-paths";
import {
  gates,
  requireGuest,
  requireSession,
  type RouteGate,
} from "@trenova/shared/lib/router/gates";
import { redirect } from "@tanstack/react-router";

// A cloud organization that has not finished the welcome wizard is sent to it from every
// signed-in page; the wizard's own path is exempt, so this cannot loop.
const onboardingRedirect: RouteGate = async ({ location }) => {
  const target = await resolveOnboardingRedirect(location.pathname);
  if (target) {
    throw redirect({ href: target });
  }
};

/**
 * The gate every signed-in layout runs: the session, the onboarding redirect, then the
 * edition's own gates, the first to throw deciding. Each layout gets its own, because a gate
 * remembers the path it last let through (see `gates`).
 */
export function protectedGate(): RouteGate {
  return gates(requireSession(webSignInPaths), onboardingRedirect, ...edition.protectedGates);
}

// The wizard is only for an organization that still owes it. Anybody else who opens
// the address — a completed organization, a self-hosted install — goes home.
export const onboardingGate: RouteGate = async () => {
  const target = onboardingExitRedirect(await loadOnboardingSnapshot());
  if (target) {
    throw redirect({ href: target });
  }
};

export const guestGate = requireGuest(webSignInPaths.home);

// A tender's record link opens the inbound page; one we sent is carried to the outbound page.
export const ediSentTransferGate: RouteGate = async ({ location, abortController }) => {
  const url = new URL(`${location.pathname}${location.searchStr}`, window.location.origin);
  if (!url.searchParams.has("panelEntityId")) {
    return;
  }
  const { sentTransferPath } = await import("@/lib/edi/transfer-page");
  const sent = await sentTransferPath(url, abortController.signal);
  if (sent) {
    throw redirect({ href: sent });
  }
};

// A layout entered for the first time (the app's first load, or arriving from sign-in) draws
// its loading screen at once, as the page used to before the router had answered. A layout
// that stays while its page changes is already loaded, so a change of page keeps the page
// standing with the progress bar until the next one is ready.
export const SHOWN_AT_ONCE = { pendingMs: 0, pendingMinMs: 0 } as const;
