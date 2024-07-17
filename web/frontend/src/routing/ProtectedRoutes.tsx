/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
