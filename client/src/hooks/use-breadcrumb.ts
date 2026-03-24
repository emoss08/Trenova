import { generateBreadcrumbSegments, getPageTitle } from "@/lib/route-utils";
import type { BreadcrumbMatch } from "@/types/router";
import { useLocation, useMatches } from "react-router";

export function useBreadcrumbs() {
  const matches = useMatches() as BreadcrumbMatch[];
  const location = useLocation();

  // Get matches with explicit crumbs from route handles
  const explicitCrumbs = matches
    .filter(
      (match) => match.handle?.crumb && match.handle?.showBreadcrumbs !== false,
    )
    .map((match) => ({
      id: match.id,
      pathname: match.pathname,
      crumb:
        typeof match.handle?.crumb === "function"
          ? match.handle.crumb(match.data)
          : match.handle?.crumb,
    }));

  // Generate implicit crumbs from the current path
  const segments = generateBreadcrumbSegments(location.pathname);

  // Merge explicit and implicit crumbs, preferring explicit ones
  const allCrumbs = segments.map((segment, index) => {
    const explicitCrumb = explicitCrumbs.find(
      (c) => c.pathname === segment.path,
    );

    const isLastSegment = index === segments.length - 1;
    return (
      explicitCrumb || {
        id: segment.path,
        pathname: segment.path,
        crumb: isLastSegment ? getPageTitle(segment.path) : segment.label,
      }
    );
  });

  return allCrumbs;
}
