import { organization } from "@/lib/queries/organization";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef } from "react";

/**
 * How long a read of the branding is trusted. The addresses it returns are signed by storage
 * and lapse (fifteen minutes by default), so they are read again well inside that.
 */
const BRANDING_FRESH_MS = 10 * 60 * 1000;

/**
 * Where an organization's logo and icon are read from. The addresses are short-lived, so a
 * screen that stays open reads them again on a timer and when its tab is returned to, and a
 * picture that fails to load says so with `refresh`, which reads them again at once: a
 * picture drawn again after its address lapsed would otherwise be a broken image until the
 * page was reloaded.
 */
export function useOrganizationBranding(organizationId: string | null | undefined, enabled = true) {
  const queryClient = useQueryClient();
  const refreshed = useRef<string | null>(null);

  const { data: branding } = useQuery({
    ...organization.branding(organizationId ?? ""),
    enabled: enabled && Boolean(organizationId),
    retry: false,
    staleTime: BRANDING_FRESH_MS,
    refetchInterval: BRANDING_FRESH_MS,
    refetchOnWindowFocus: true,
  });

  /**
   * Reads the addresses again after the picture at `failedUrl` did not load. Each address is
   * given one new read, so a picture that is truly gone does not ask forever.
   */
  const refresh = (failedUrl: string) => {
    if (!organizationId || refreshed.current === failedUrl) return;
    refreshed.current = failedUrl;
    void queryClient.invalidateQueries({
      queryKey: organization.branding(organizationId).queryKey,
    });
  };

  return { branding, refresh };
}
