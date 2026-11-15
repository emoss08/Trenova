import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { gates, redirectTo, requireGuest, requireSession } from "@trenova/shared/lib/router/gates";
import {
  parseSearch,
  stringifySearch,
  validateSearchRecord,
} from "@trenova/shared/lib/router/search";
import {
  createRootRoute,
  createRoute,
  createRouter,
  lazyRouteComponent,
  type ErrorComponentProps,
} from "@tanstack/react-router";
import { DashLayout } from "./_components/dash-layout";

function DashRouteError(props: ErrorComponentProps) {
  return <RouteErrorBoundary {...props} homePath="/dash" />;
}

const rootRoute = createRootRoute({
  validateSearch: validateSearchRecord,
  pendingComponent: LoadingSkeleton,
  pendingMs: 0,
});

const signedInRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: "signed-in",
  beforeLoad: gates(requireSession("/dash/login")),
  component: DashLayout,
});

export const routeTree = rootRoute.addChildren([
  createRoute({
    getParentRoute: () => rootRoute,
    path: "/dash/login",
    beforeLoad: gates(requireGuest("/dash")),
    component: lazyRouteComponent(() => import("./routes/login"), "DashLoginPage"),
  }),
  createRoute({
    getParentRoute: () => rootRoute,
    path: "/dash/accept",
    component: lazyRouteComponent(() => import("./routes/accept"), "DashAcceptPage"),
  }),
  signedInRoute.addChildren([
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash",
      component: lazyRouteComponent(() => import("./routes/home"), "DashHomePage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/loads",
      component: lazyRouteComponent(() => import("./routes/loads"), "DashLoadsPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/loads/$assignmentId",
      component: lazyRouteComponent(() => import("./routes/load-detail"), "DashLoadDetailPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/pay",
      component: lazyRouteComponent(() => import("./routes/pay"), "DashPayPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/pay/$settlementId",
      component: lazyRouteComponent(() => import("./routes/settlement"), "DashSettlementPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/money",
      component: lazyRouteComponent(() => import("./routes/money"), "DashMoneyPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/hos",
      component: lazyRouteComponent(() => import("./routes/hos"), "DashHosPage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/notifications",
      component: lazyRouteComponent(
        () => import("./routes/notifications"),
        "DashNotificationsPage",
      ),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/profile",
      component: lazyRouteComponent(() => import("./routes/profile"), "DashProfilePage"),
    }),
    createRoute({
      getParentRoute: () => signedInRoute,
      path: "/dash/$",
      beforeLoad: redirectTo("/dash"),
    }),
  ]),
]);

export const dashRouter = createRouter({
  routeTree,
  parseSearch,
  stringifySearch,
  defaultErrorComponent: DashRouteError,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof dashRouter;
  }
}
