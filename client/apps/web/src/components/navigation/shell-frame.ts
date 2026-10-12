import type { BreadcrumbMatch, ShellFrame } from "@/types/router";
import { createContext, useContext } from "react";
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
