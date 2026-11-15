import { tanstackRouter } from "@tanstack/router-plugin/vite";
import path from "node:path";
import { ROUTE_SHELLS } from "./route-shells.mjs";

const SHELLS = new Set(ROUTE_SHELLS);

/**
 * The route tree: the files under src/pages, generated into src/route-tree.gen.ts. Each route's
 * loader and page are split into one chunk, fetched together when the route is first entered or
 * a link to it is hovered, so the loader's prefetch never waits on a second download, the entry
 * bundle holds only the routes' gates, and a test that imports the tree loads a page's code only
 * when it enters that page.
 */
export function fileRoutes({ srcDir }: { srcDir: string }) {
  return tanstackRouter({
    target: "react",
    routesDirectory: path.join(srcDir, "pages"),
    generatedRouteTree: path.join(srcDir, "route-tree.gen.ts"),
    quoteStyle: "double",
    semicolons: true,
    autoCodeSplitting: true,
    codeSplittingOptions: {
      defaultBehavior: [["loader", "component"], ["errorComponent"], ["notFoundComponent"]],
      splitBehavior: ({ routeId }) => (SHELLS.has(routeId) ? [] : undefined),
    },
  });
}
