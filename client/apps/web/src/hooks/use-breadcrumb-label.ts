import { normalizePath } from "@/lib/route-utils";
import { useBreadcrumbStore } from "@/stores/breadcrumb-store";
import { useEffect } from "react";
import { useLocation } from "react-router";

/**
 * Names the breadcrumb for the current page (or the crumb at `path`) with
 * something the URL cannot supply, like the record's name. The label is
 * published while it is truthy and withdrawn when it changes or the page
 * unmounts, so a stale name never lingers on the next route.
 */
export function useBreadcrumbLabel(label: string | null | undefined, path?: string): void {
  const { pathname } = useLocation();
  const target = normalizePath(path ?? pathname);
  const setLabel = useBreadcrumbStore((state) => state.setLabel);
  const clearLabel = useBreadcrumbStore((state) => state.clearLabel);

  useEffect(() => {
    const trimmed = label?.trim();
    if (!target || !trimmed) {
      return;
    }
    setLabel(target, trimmed);
    return () => clearLabel(target, trimmed);
  }, [target, label, setLabel, clearLabel]);
}
