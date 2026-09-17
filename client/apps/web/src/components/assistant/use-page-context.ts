import type { AssistantPageContext } from "@/types/assistant";
import { useCallback } from "react";
import { useLocation } from "react-router";
import { derivePageContext } from "./page-context";

/**
 * A getter rather than a value: the title is read at send time, after the
 * page's metadata has settled, and a getter costs nothing on every render.
 */
export function usePageContext(): () => AssistantPageContext | null {
  const { pathname, search } = useLocation();

  return useCallback(
    () =>
      derivePageContext({
        pathname,
        search,
        title: typeof document === "undefined" ? "" : document.title,
      }),
    [pathname, search],
  );
}
