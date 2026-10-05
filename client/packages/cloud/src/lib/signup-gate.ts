import { queryClient } from "@/lib/query-client";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { isSignupAvailable, type PublicConfig } from "@trenova/shared/types/platform";

export const SIGNUP_PATH = "/signup";

/** Where a signup page request goes instead, or null when signup is open. */
export function signupRedirect(
  config: Pick<PublicConfig, "platformMode" | "signupEnabled"> | undefined,
): string | null {
  return isSignupAvailable(config) ? null : "/login";
}

export async function signupRouteRedirect(): Promise<string | null> {
  return signupRedirect(await queryClient.ensureQueryData(publicConfigQueryOptions));
}
