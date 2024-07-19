/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import { FavoriteIcon } from "./user-favorite";

const useRouteMatching = (
  setLoading: (loading: boolean) => void,
  setCurrentRoute: (route: any) => void,
) => {
  const location = useLocation();

  useEffect(() => {
    setLoading(true);
    const matchedRoute = routes.find((route) => {
      return (
        route.path !== "*" && pathToRegexp(route.path).test(location.pathname)
      );
    });

    setCurrentRoute(matchedRoute || null);
    setLoading(false);
  }, [location, setLoading, setCurrentRoute]);
};

const useDocumentTitle = (currentRoute: any) => {
  useEffect(() => {
    if (currentRoute) {
      document.title = currentRoute.title;
    }
  }, [currentRoute]);
};

export function Breadcrumb({ children }: { children?: React.ReactNode }) {
  const [currentRoute, setCurrentRoute] =
    useBreadcrumbStore.use("currentRoute");
  const [loading, setLoading] = useBreadcrumbStore.use("loading");
  useRouteMatching(setLoading, setCurrentRoute);
  useDocumentTitle(currentRoute);

  // Construct breadcrumb text
  const breadcrumbText = useMemo(() => {
    if (!currentRoute) return "";
    const parts = [currentRoute.group, currentRoute.subMenu, currentRoute.title]
      .filter((str): str is string => str !== undefined)
      .map(upperFirst);
    return parts.join(" - ");
  }, [currentRoute]);

  if (loading) {
    return (
      <>
        <Skeleton className="h-[30px] w-[200px]" />
        <Skeleton className="mt-5 h-[30px] w-[200px]" />
      </>
    );
  }

  // If the current route is not found or is an excluded path, return null
  if (!currentRoute) {
    return null;
  }

  return (
    <div className="pb-4 pt-5 md:py-4">
      <div>
        <h2 className="mt-10 flex scroll-m-20 items-center pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          {currentRoute?.title}
          <FavoriteIcon />
        </h2>
        <div className="flex items-center">
          <a className="text-sm font-medium text-muted-foreground hover:text-muted-foreground/80">
            {breadcrumbText}
          </a>
        </div>
      </div>
      <div className="mt-3 flex">{children}</div>
    </div>
  );
}
