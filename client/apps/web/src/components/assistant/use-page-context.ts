import { readPageView } from "@/stores/page-view-store";
import type { AssistantPageContext } from "@/types/assistant";
import { useCallback } from "react";
import { useLocation } from "react-router";
import { derivePageContext } from "./page-context";

/**
 * A getter rather than a value: the title is read at send time, after the
 * page's metadata has settled, and a getter costs nothing on every render.
 * The table the page is showing is read the same way, so the filters sent
 * are the ones on screen when the question leaves.
 */
export function usePageContext(): () => AssistantPageContext | null {
  const { pathname, search } = useLocation();

  return useCallback(() => {
    const context = derivePageContext({
      pathname,
      search,
      title: typeof document === "undefined" ? "" : document.title,
    });
    if (context === null) {
      return null;
    }
    const view = readPageView();

    return view === null ? context : { ...context, view };
  }, [pathname, search]);
}
