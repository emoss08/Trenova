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

import React, { useEffect } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router-dom";
import { Layout } from "@/components/layout/Layout";
import { RouteObjectWithPermission, routes } from "./AppRoutes";
import { useUserPermissions } from "@/hooks/useUserPermissions";

export function ProtectedRoutes(): React.ReactElement {
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const location = useLocation();

  useEffect(() => {
    if (
      !isAuthenticated &&
      location.pathname !== "/login"
      // location.pathname !== "/logout"
    ) {
      const returnUrl = location.pathname + location.search;
      sessionStorage.setItem("returnUrl", returnUrl);
    }
  }, [isAuthenticated, location.pathname, location.search]);

  return (
    <Routes>
      {routes.map((route: RouteObjectWithPermission, i: number) => {
        const isPublicRoute =
          route.path === "/login" ||
          // route.path === "/logout" ||
          route.path === "/reset-password";

        let element: React.ReactNode;
        if (isPublicRoute || isAuthenticated) {
          if (route.permission) {
            if (userHasPermission(route.permission)) {
              element = route.element;
            } else {
              element = <Navigate to="/error" replace />;
            }
          } else {
            element = route.element;
          }
        } else {
          element = <Navigate to="/login" replace />;
        }

        const wrappedElement = isPublicRoute ? (
          element
        ) : (
          <Layout>{element}</Layout>
        );

        return <Route key={i} path={route.path} element={wrappedElement} />;
      })}
    </Routes>
  );
}
