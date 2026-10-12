import { DeskWorkspaceSkeleton } from "@/components/desk-chat/desk-skeletons";
import { preloadable } from "@/lib/lazy-component";
import { lazy, Suspense, type ComponentProps } from "react";

/**
 * Reads the workspace's code ahead of its first opening: when a conversation
 * has something to show in it, or the pointer reaches its button. Repeat calls
 * share the read; a failed one is forgotten so the next asks again.
 */
export const preloadDeskWorkspace = preloadable(() => import("./desk-workspace"));

const Workspace = lazy(() =>
  preloadDeskWorkspace().then((module) => ({ default: module.DeskWorkspace })),
);

/**
 * The workspace, whose code is read only once it is first wanted: it is
 * closed in most conversations, and its artifact views are most of the
 * conversation page's weight. Its outline shows while the code arrives.
 */
export function LazyDeskWorkspace(props: ComponentProps<typeof Workspace>) {
  return (
    <Suspense fallback={<DeskWorkspaceSkeleton />}>
      <Workspace {...props} />
    </Suspense>
  );
}
