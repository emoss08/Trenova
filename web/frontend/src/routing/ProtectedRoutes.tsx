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

import { Layout, UnprotectedLayout } from "@/components/layout/layout";
import { useUserPermissions } from "@/context/user-permissions";
import { useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { RouteObjectWithPermission, routes } from "./AppRoutes";

export function ProtectedRoutes() {
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const location = useLocation();

  useEffect(() => {
    if (
      !isAuthenticated &&
      !["/login", "/reset-password"].includes(location.pathname)
    ) {
      const returnUrl = location.pathname + location.search;
      sessionStorage.setItem("returnUrl", returnUrl);
    }
  }, [isAuthenticated, location.pathname, location.search]);

  const getElement = (route: RouteObjectWithPermission): React.ReactNode => {
    if (!isAuthenticated && !route.isPublic) {
      return <Navigate to="/login" replace />;
    }
    if (route.permission && !userHasPermission(route.permission)) {
      console.info(
        `User does not have permission: ${route.permission} for route: ${route.path}`,
      );
      return <Navigate to="/error" replace />;
    }
    return route.element ?? null;
  };

  const getLayout = (
    route: RouteObjectWithPermission,
    element: React.ReactNode,
  ): JSX.Element => {
    return route.isPublic ? (
      <UnprotectedLayout>{element}</UnprotectedLayout>
    ) : (
      <Layout>{element}</Layout>
    );
  };

  return (
    <Routes>
      {routes.map((route: RouteObjectWithPermission) => {
        const element = getElement(route);
        const wrappedElement = getLayout(route, element);
        return (
          <Route
            key={route.key || route.path}
            path={route.path}
            element={wrappedElement}
          />
        );
      })}
    </Routes>
  );
}
