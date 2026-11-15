import { RootLayout } from "@/routes/root-layout";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { validateSearchRecord } from "@trenova/shared/lib/router/search";
import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext } from "@tanstack/react-router";

/** What every route's gates and loader are handed by the router. */
export interface RouterContext {
  readonly queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
  validateSearch: validateSearchRecord,
  pendingComponent: LoadingSkeleton,
  pendingMs: 0,
});
