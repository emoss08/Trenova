import { generateBreadcrumbSegments, type BreadcrumbSegment } from "@/lib/route-utils";
import { useBreadcrumbStore } from "@/stores/breadcrumb-store";
import { useMemo } from "react";
import { useLocation } from "react-router";

/** The crumbs for the current location, named by the pages that published labels. */
export function useBreadcrumbs(): BreadcrumbSegment[] {
  const { pathname } = useLocation();
  const labels = useBreadcrumbStore((state) => state.labels);

  return useMemo(() => generateBreadcrumbSegments(pathname, labels), [pathname, labels]);
}
