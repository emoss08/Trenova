/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { upperFirst } from "@/lib/utils";
import { routes } from "@/routing/AppRoutes";
import { useBreadcrumbStore } from "@/stores/BreadcrumbStore";
import { pathToRegexp } from "path-to-regexp";
import { useEffect, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { Skeleton } from "../ui/skeleton";

export function Breadcrumb() {
  const location = useLocation();
  const [currentRoute, setCurrentRoute] =
    useBreadcrumbStore.use("currentRoute");
  const [loading, setLoading] = useBreadcrumbStore.use("loading");

  // Find the matching route based on the current pathname
  useEffect(() => {
    setLoading(true);

    const matchedRoute = routes.find((r) => {
      if (r.path === "*") return false;
      return pathToRegexp(r.path).test(location.pathname);
    });

    if (matchedRoute) {
      setCurrentRoute(matchedRoute);
    }

    setLoading(false);
  }, [location.pathname, setCurrentRoute, setLoading]);

  // Update document title when the current route changes
  useEffect(() => {
    if (currentRoute) {
      document.title = currentRoute.title;
    }
  }, [currentRoute]);

  // Construct breadcrumb text, memoized to avoid recalculations
  const breadcrumbText = useMemo(() => {
    if (!currentRoute) return "";
    const parts = [
      currentRoute.group,
      currentRoute.subMenu,
      currentRoute.title,
    ].filter((str: string | undefined): str is string => Boolean(str));

    return parts.map(upperFirst).join(" - ");
  }, [currentRoute]);

  return loading ? (
    <>
      <Skeleton className="w-[200px] h-[30px]" />
      <Skeleton className="w-[200px] h-[30px] mt-5" />
    </>
  ) : (
    <div className=" pt-5 pb-4 md:pt-4 md:pb-4">
      <h2 className="mt-10 scroll-m-20 pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
        {currentRoute?.title}
      </h2>
      <div className="flex items-center">
        <a className="text-sm font-medium text-gray-500 hover:text-gray-700">
          {breadcrumbText}
        </a>
      </div>
    </div>
  );
}
