import { ErrorStateBoundary } from "@trenova/shared/components/error-boundary";
import { Outlet } from "react-router";

/**
 * Nothing suspends at this outlet, so it catches failures only. Each admin page's code is read
 * by the router's `lazy` before the route commits, inside a transition that keeps the previous
 * page on screen, and every admin page draws its own skeleton around what it loads. A Suspense
 * here could only ever show one shape for forty different pages.
 *
 * It draws nothing of its own: the page sits straight in the shell's scrolling frame, as a
 * page outside the admin area does, so one that fills the frame (`PageLayout fill`, a data
 * table) can take its height. A wrapper here, with a height of its own content, left every
 * admin page unable to.
 */
export function AdminLayout() {
  return (
    <ErrorStateBoundary>
      <Outlet />
    </ErrorStateBoundary>
  );
}
