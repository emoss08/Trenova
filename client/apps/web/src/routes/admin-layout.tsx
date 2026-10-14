import { ErrorStateBoundary } from "@trenova/shared/components/error-boundary";
import { Outlet } from "react-router";

/**
 * Nothing suspends at this outlet, so it catches failures only. Each admin page's code is read
 * by the router's `lazy` before the route commits, inside a transition that keeps the previous
 * page on screen, and every admin page draws its own skeleton around what it loads. A Suspense
 * here could only ever show one shape for forty different pages.
 */
export function AdminLayout() {
  return (
    <div className="flex-1">
      <div className="mx-auto min-w-0">
        <ErrorStateBoundary>
          <Outlet />
        </ErrorStateBoundary>
      </div>
    </div>
  );
}
