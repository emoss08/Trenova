import { assistant } from "@/lib/queries/assistant";
import { useQuery } from "@tanstack/react-query";

/**
 * How long the answer holds before a composer that mounts asks again. Turning
 * an extension on or off invalidates it at once (extension-detail.tsx); this
 * covers a change made on another device.
 */
const BROWSER_STALE_MS = 60_000;

/**
 * Whether /browser is offered for an agent: only while portal automation is
 * on, the agent holds its tools or every agent is given them, and the person
 * may use it. The server decides, so the command stays hidden until it has
 * answered, and when it cannot.
 */
export function useDeskBrowser(agentId: string | null | undefined): boolean {
  const { data } = useQuery({
    ...assistant.browser(agentId ?? ""),
    enabled: agentId !== null && agentId !== undefined && agentId !== "",
    staleTime: BROWSER_STALE_MS,
    refetchOnWindowFocus: true,
    retry: false,
  });

  return data?.available ?? false;
}
