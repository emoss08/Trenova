/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { generateBreadcrumbSegments } from "@/lib/route-utils";
import { type BreadcrumbMatch } from "@/types/router";
import { useLocation, useMatches } from "react-router-dom";

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
  const allCrumbs = segments.map((segment) => {
    const explicitCrumb = explicitCrumbs.find(
      (c) => c.pathname === segment.path,
    );
    return (
      explicitCrumb || {
        id: segment.path,
        pathname: segment.path,
        crumb: segment.label,
      }
    );
  });

  return allCrumbs;
}
