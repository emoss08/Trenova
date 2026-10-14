import type { BreadcrumbMatch, ShellFrame } from "@/types/router";
import { createContext, useContext, type ReactNode } from "react";
import { useMatches } from "react-router";

/**
 * The frame the shell drew around the current route, for the page inside it.
 * Outside the shell (a test, a story) it is a page.
 */
export const ShellFrameContext = createContext<ShellFrame>("page");

export function useShellFrame(): ShellFrame {
  return useContext(ShellFrameContext);
}

/** The frame the deepest route that names one asks for; a page when none does. */
export function frameOfMatches(matches: readonly Pick<BreadcrumbMatch, "handle">[]): ShellFrame {
  for (let index = matches.length - 1; index >= 0; index -= 1) {
    const frame = matches[index].handle?.frame;
    if (frame !== undefined) {
      return frame;
    }
  }

  return "page";
}

/** The frame the matched routes ask the shell for. */
export function useRouteFrame(): ShellFrame {
  const matches = useMatches() as unknown as BreadcrumbMatch[];

  return frameOfMatches(matches);
}

/**
 * What the shell puts on a page's title bar around the page's own content:
 * the fold, history and the trail before the title, the star after the
 * actions. The shell provides it and `PageLayout` draws it, so a page never
 * reaches for the router's data itself and renders the same outside the
 * shell, where there is nothing to lead with.
 */
export interface PageBarSlots {
  leading: (title: string) => ReactNode;
  trailing: ReactNode;
}

export const PageBarSlotsContext = createContext<PageBarSlots | null>(null);

export function usePageBarSlots(): PageBarSlots | null {
  return useContext(PageBarSlotsContext);
}
