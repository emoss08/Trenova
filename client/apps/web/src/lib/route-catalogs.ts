import { afterCatalogs } from "@trenova/shared/i18n/runtime";
import type { RouteComponent } from "@tanstack/react-router";
import { createElement, use, type ComponentType } from "react";

/** A route's page component, with the load the router awaits before it first draws it. */
export type LazyPage = { preload: () => Promise<void> } & RouteComponent;

/**
 * A route's page, loaded when the route is first visited. The router awaits `preload` before it
 * draws the page, and a failed import rejects it, so the route's error component presents the
 * failure (a release that replaced the page's file reads as a new version, with a retry) and
 * chunk recovery decides whether to reload (lib/chunk-recovery: only for a newer build, never
 * over unsaved work). The router's own lazy component would reload the window by itself.
 *
 * A route's strings ship in its folder's catalog bundle, which every module in the folder
 * requires as it is evaluated (vite/route-catalogs.ts injects the call). By the time the page's
 * import resolves, its whole module graph has evaluated, so every bundle the page can render
 * from has been asked for — including those of other route folders it borrows components from.
 * Waiting for them before the page resolves is what lets its first frame render translated
 * instead of in English and then again.
 */
export function lazyPage<
  TModule extends Record<string, unknown>,
  TExport extends keyof TModule & string,
>(importer: () => Promise<TModule>, exportName: TExport): LazyPage {
  let Page: ComponentType<object> | undefined;
  let loading: Promise<void> | undefined;

  const preload = (): Promise<void> => {
    loading ??= importer()
      .then(afterCatalogs)
      .then(
        (module) => {
          Page = module[exportName] as ComponentType<object>;
        },
        (error: unknown) => {
          // A retry imports again.
          loading = undefined;
          throw error;
        },
      );
    return loading;
  };

  function LazyPage(props: object) {
    if (!Page) {
      use(preload());
    }
    return Page ? createElement(Page, props) : null;
  }
  LazyPage.preload = preload;

  return LazyPage;
}
