import { useMediaQuery } from "@/hooks/use-media-query";
import { useNavigationStore } from "@/stores/navigation-store";
import { useCallback } from "react";

/**
 * Below this width the sidebar never sits beside the page: a phone has no
 * room for both, so it slides over the page as a sheet and starts closed.
 * The same breakpoint as Tailwind's `md`, which the shell's classes use.
 */
export const SHELL_COMPACT_QUERY = "(max-width: 767px)";

/**
 * Whether the sidebar is showing, and the one way to show or hide it, for the
 * toggle and for ⌘B alike. Beside the page that is the kept fold; on a phone
 * it is the sheet, which is never kept.
 */
export function useSidebarFold() {
  const compact = useMediaQuery(SHELL_COMPACT_QUERY);
  const collapsed = useNavigationStore((state) => state.sidebarCollapsed);
  const compactOpen = useNavigationStore((state) => state.compactSidebarOpen);
  const toggleSidebarFor = useNavigationStore((state) => state.toggleSidebarFor);
  const toggle = useCallback(() => toggleSidebarFor(compact), [toggleSidebarFor, compact]);

  return { compact, shown: compact ? compactOpen : !collapsed, toggle };
}
