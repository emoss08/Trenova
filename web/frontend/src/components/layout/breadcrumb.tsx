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
import { useEffect, useMemo } from "react";
import { Link, matchPath, useLocation } from "react-router-dom";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "../ui/breadcrumb";
import { Skeleton } from "../ui/skeleton";
import { FavoriteIcon } from "./user-favorite";

type BreadcrumbItemType = {
  label: string;
  path: string;
};

export function SiteBreadcrumb({ children }: { children?: React.ReactNode }) {
  const location = useLocation();
  const [currentRoute, setCurrentRoute] =
    useBreadcrumbStore.use("currentRoute");
  const [loading, setLoading] = useBreadcrumbStore.use("loading");

  const matchingRoute = routes.find(
    (route) => route.path !== "*" && matchPath(route.path, location.pathname),
  );

  useEffect(() => {
    setLoading(true);
    const matchedRoute = matchingRoute;
    setCurrentRoute(matchedRoute || null);
    setLoading(false);
  }, [location, setCurrentRoute, setLoading, matchingRoute]);

  useEffect(() => {
    if (currentRoute) {
      document.title = currentRoute.title;
    }
  }, [currentRoute]);

  const breadcrumbItems = useMemo(() => {
    if (!currentRoute) return [];
    const items: BreadcrumbItemType[] = [
      { label: "Home", path: "/" },
      ...(currentRoute.group
        ? [
            {
              label: upperFirst(currentRoute.group),
              path: `/${currentRoute.group}`,
            },
          ]
        : []),
      ...(currentRoute.subMenu
        ? [
            {
              label: upperFirst(currentRoute.subMenu),
              path: `/${currentRoute.group}/${currentRoute.subMenu}`,
            },
          ]
        : []),
      { label: currentRoute.title, path: location.pathname },
    ];
    return items;
  }, [currentRoute, location.pathname]);

  if (loading) {
    return (
      <>
        <Skeleton className="h-[30px] w-[200px]" />
        <Skeleton className="mt-5 h-[30px] w-[200px]" />
      </>
    );
  }

  if (!currentRoute) {
    return null;
  }

  return (
    <div className="pb-4 pt-5 md:py-4">
      <div>
        <h2 className="mt-10 flex scroll-m-20 items-center pb-2 text-xl font-semibold tracking-tight transition-colors first:mt-0">
          {currentRoute.title}
          <FavoriteIcon />
        </h2>
        <Breadcrumb>
          <BreadcrumbList>
            {breadcrumbItems.map((item, index) => (
              <BreadcrumbItem key={item.path}>
                {index === breadcrumbItems.length - 1 ? (
                  <BreadcrumbPage>{item.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link to={item.path}>{item.label}</Link>
                  </BreadcrumbLink>
                )}
                {index < breadcrumbItems.length - 1 && <BreadcrumbSeparator />}
              </BreadcrumbItem>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      </div>
      <div className="mt-3 flex">{children}</div>
    </div>
  );
}
