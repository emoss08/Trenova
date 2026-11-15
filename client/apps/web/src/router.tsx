import { mountEditionRoutes } from "@/lib/edition-routes";
import { queryClient } from "@/lib/query-client";
import { routeTree } from "@/route-tree.gen";
import { RouteErrorBoundary } from "@trenova/shared/components/error-boundary";
import { parseSearch, stringifySearch } from "@trenova/shared/lib/router/search";
import { createRouter } from "@tanstack/react-router";

mountEditionRoutes();

export const router = createRouter({
  routeTree,
  context: { queryClient },
  parseSearch,
  stringifySearch,
  // Hovering or focusing a link loads the page's chunk and starts its prefetch, so the click
  // lands on a page that is already on its way. Gates that would ask the server skip a preload.
  defaultPreload: "intent",
  // Route loaders only warm TanStack Query, which decides freshness itself.
  defaultPreloadStaleTime: 0,
  // A route without an error component of its own draws the failure where its page would
  // have been; inside the shell or the Desk the frame stays standing around it.
  defaultErrorComponent: RouteErrorBoundary,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
