import { onboardingStateQueryOptions } from "@/lib/queries/onboarding";
import { queryClient } from "@/lib/query-client";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { isCloudPlatform, type PublicConfig } from "@trenova/shared/types/platform";
import type { OnboardingState } from "@/types/onboarding";

export const ONBOARDING_PATH = "/onboarding";

function normalizePathname(pathname: string): string {
  if (pathname === "/") {
    return pathname;
  }
  return pathname.replace(/\/+$/, "");
}

export function isOnboardingPath(pathname: string): boolean {
  const normalized = normalizePathname(pathname);
  return normalized === ONBOARDING_PATH || normalized.startsWith(`${ONBOARDING_PATH}/`);
}

/** Whether this organization still owes the wizard. Only cloud organizations ever do. */
export function onboardingPending(
  config: Pick<PublicConfig, "platformMode"> | undefined,
  state: Pick<OnboardingState, "required" | "status"> | null | undefined,
): boolean {
  return isCloudPlatform(config) && state?.required === true && state.status === "pending";
}

/**
 * Where a signed-in page request has to go instead, or null to let it through. The
 * wizard itself is never redirected, so a pending organization cannot loop.
 */
export function onboardingRedirect({
  pathname,
  config,
  state,
}: {
  pathname: string;
  config: Pick<PublicConfig, "platformMode"> | undefined;
  state: Pick<OnboardingState, "required" | "status"> | null | undefined;
}): string | null {
  if (isOnboardingPath(pathname)) {
    return null;
  }
  return onboardingPending(config, state) ? ONBOARDING_PATH : null;
}

/** Where the wizard sends somebody who has nothing left to do in it. */
export function onboardingExitRedirect({
  config,
  state,
}: {
  config: Pick<PublicConfig, "platformMode"> | undefined;
  state: Pick<OnboardingState, "required" | "status"> | null | undefined;
}): string | null {
  return onboardingPending(config, state) ? null : "/";
}

export type OnboardingSnapshot = {
  config: PublicConfig;
  state: OnboardingState | null;
};

/**
 * Reads what the gate needs from the query cache, fetching only what is missing. The
 * public config never throws (it falls back to self-hosted), and a failed onboarding
 * read lets the request through: the wizard is a convenience the API does not depend
 * on, so an outage there must not lock anybody out of the product.
 */
export async function loadOnboardingSnapshot(): Promise<OnboardingSnapshot> {
  const config = await queryClient.ensureQueryData(publicConfigQueryOptions);
  if (!isCloudPlatform(config)) {
    return { config, state: null };
  }

  const organizationId = useAuthStore.getState().user?.currentOrganizationId ?? "";
  try {
    const state = await queryClient.ensureQueryData(onboardingStateQueryOptions(organizationId));
    return { config, state };
  } catch {
    return { config, state: null };
  }
}

export async function resolveOnboardingRedirect(pathname: string): Promise<string | null> {
  const { config, state } = await loadOnboardingSnapshot();
  return onboardingRedirect({ pathname, config, state });
}
