import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { requireSession, revalidateOnPathChange } from "@trenova/shared/lib/session-loader";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { createBrowserRouter, redirect, type LoaderFunction, type RouteObject } from "react-router";
import { DashLayout } from "./_components/dash-layout";

const dashProtectedLoader: LoaderFunction = () => requireSession("/dash/login");

const dashGuestLoader: LoaderFunction = async () => {
  const { checkAuth } = useAuthStore.getState();
  const isAuthenticated = await checkAuth();

  if (isAuthenticated) {
    return redirect("/dash");
  }

  return null;
};

export const routes: RouteObject[] = [
  {
    errorElement: <RouteErrorBoundary homePath="/dash" />,
    HydrateFallback: LoadingSkeleton,
    children: [
      {
        path: "/dash/login",
        loader: dashGuestLoader,
        async lazy() {
          const { DashLoginPage } = await import("./routes/login");
          return { Component: DashLoginPage };
        },
      },
      {
        path: "/dash/accept",
        async lazy() {
          const { DashAcceptPage } = await import("./routes/accept");
          return { Component: DashAcceptPage };
        },
      },
      {
        element: <DashLayout />,
        loader: dashProtectedLoader,
        shouldRevalidate: revalidateOnPathChange,
        children: [
          {
            path: "/dash",
            async lazy() {
              const { DashHomePage } = await import("./routes/home");
              return { Component: DashHomePage };
            },
          },
          {
            path: "/dash/loads",
            async lazy() {
              const { DashLoadsPage } = await import("./routes/loads");
              return { Component: DashLoadsPage };
            },
          },
          {
            path: "/dash/loads/:assignmentId",
            async lazy() {
              const { DashLoadDetailPage } = await import("./routes/load-detail");
              return { Component: DashLoadDetailPage };
            },
          },
          {
            path: "/dash/pay",
            async lazy() {
              const { DashPayPage } = await import("./routes/pay");
              return { Component: DashPayPage };
            },
          },
          {
            path: "/dash/pay/:settlementId",
            async lazy() {
              const { DashSettlementPage } = await import("./routes/settlement");
              return { Component: DashSettlementPage };
            },
          },
          {
            path: "/dash/money",
            async lazy() {
              const { DashMoneyPage } = await import("./routes/money");
              return { Component: DashMoneyPage };
            },
          },
          {
            path: "/dash/hos",
            async lazy() {
              const { DashHosPage } = await import("./routes/hos");
              return { Component: DashHosPage };
            },
          },
          {
            path: "/dash/notifications",
            async lazy() {
              const { DashNotificationsPage } = await import("./routes/notifications");
              return { Component: DashNotificationsPage };
            },
          },
          {
            path: "/dash/profile",
            async lazy() {
              const { DashProfilePage } = await import("./routes/profile");
              return { Component: DashProfilePage };
            },
          },
          {
            path: "/dash/*",
            loader: () => redirect("/dash"),
          },
        ],
      },
    ],
  },
];

export const dashRouter = createBrowserRouter(routes);
