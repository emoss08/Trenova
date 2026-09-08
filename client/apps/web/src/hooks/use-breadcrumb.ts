import { generateBreadcrumbSegments } from "@/lib/route-utils";
import { useBreadcrumbStore } from "@/stores/breadcrumb-store";
import type { BreadcrumbMatch } from "@/types/router";
import { useMemo } from "react";
import { useLocation, useMatches } from "react-router";

export function useBreadcrumbs() {
  const matches = useMatches() as unknown as BreadcrumbMatch[];
  const location = useLocation();
  const labels = useBreadcrumbStore((state) => state.labels);

  return useMemo(() => {
    // Crumbs a route declares on its handle win over anything derived from the URL
    const explicitCrumbs = matches
      .filter((match) => match.handle?.crumb && match.handle?.showBreadcrumbs !== false)
      .map((match) => ({
        id: match.id,
        pathname: match.pathname,
        crumb:
          typeof match.handle?.crumb === "function"
            ? match.handle.crumb(match.data)
            : match.handle?.crumb,
      }));

    return generateBreadcrumbSegments(location.pathname, labels).map(
      (segment) =>
        explicitCrumbs.find((crumb) => crumb.pathname === segment.path) ?? {
          id: segment.path,
          pathname: segment.path,
          crumb: segment.label,
        },
    );
  }, [matches, location.pathname, labels]);
}
