import { lazy, Suspense } from "react";

// The devtools exist only in development: Vite replaces import.meta.env.DEV with false in a
// production build, so the dynamic import below is dropped and no devtools code ships.
const TanStackRouterDevtools = import.meta.env.DEV
  ? lazy(() =>
      import("@tanstack/react-router-devtools").then((module) => ({
        default: module.TanStackRouterDevtools,
      })),
    )
  : null;

/**
 * TanStack Router's devtools, bottom-right (React Query's sit bottom-left). It reads the router
 * from the root route's context and renders nothing outside development.
 */
export function RouterDevtools() {
  if (!TanStackRouterDevtools) {
    return null;
  }

  return (
    <Suspense fallback={null}>
      <TanStackRouterDevtools position="bottom-right" />
    </Suspense>
  );
}
