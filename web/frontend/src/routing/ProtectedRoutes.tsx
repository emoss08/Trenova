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
