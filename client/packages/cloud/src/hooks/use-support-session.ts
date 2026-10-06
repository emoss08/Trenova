import { useQuery } from "@tanstack/react-query";
import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { supportAccess } from "../lib/queries/support-access";
import { SUPPORT_SESSION_POLL_MS } from "../lib/support-access";

const PROFILE_STALE_MS = 5 * 60 * 1000;

/** Whether the signed-in person is Trenova platform staff. Never asked outside Cloud. */
export function useStaffProfile() {
  const { isCloud } = usePublicConfig();
  return useQuery({
    ...supportAccess.staffProfile(),
    enabled: isCloud,
    staleTime: PROFILE_STALE_MS,
    retry: false,
  });
}

/**
 * The support session this browser is in, refreshed every half minute so a revoked
 * grant or an expired session takes the person out promptly.
 */
export function useCurrentSupportSession(enabled: boolean) {
  return useQuery({
    ...supportAccess.currentSession(),
    enabled,
    refetchInterval: SUPPORT_SESSION_POLL_MS,
    refetchOnWindowFocus: true,
    retry: false,
  });
}
