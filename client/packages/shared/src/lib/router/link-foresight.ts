import { ForesightManager } from "js.foresight";
import { whenBrowserIdle } from "@trenova/shared/lib/when-page-settles";

// How soon a link may warm its page again. The page's queries decide whether anything is
// fetched (a fresh query is not), so this only bounds how often the router runs the page's
// loader for a pointer circling the same link.
export const PAGE_REWARM_MS = 15_000;

// How long links that entered the document may wait for idle time before they are looked at.
// Short enough that a link drawn with the page is watched before the person reaches for it.
const LINK_SCAN_IDLE_TIMEOUT_MS = 500;

/**
 * The part of a TanStack router the prediction reads. Methods, so a real router (whose
 * routes and options are richer than these) is accepted as it is.
 */
export interface PredictingRouter {
  readonly options: { parseSearch(searchStr: string): Record<string, unknown> };
  getMatchedRoutes(
    pathname: string,
  ): readonly [ReadonlyArray<unknown>, Record<string, string>, unknown];
  preloadRoute(options: {
    to: string;
    search: Record<string, unknown>;
    hash: string;
  }): Promise<unknown>;
}

export interface LinkForesightOptions {
  readonly router: PredictingRouter;
  /** Whether a same-origin URL is a page of this app rather than the API, a file or another app. */
  readonly isAppPage: (url: URL) => boolean;
  /** Whether a URL's query names a record (`?panelEntityId=ord_…` opens one over a list). */
  readonly namesRecord: (url: URL) => boolean;
  /** Where links are watched; the whole document by default. */
  readonly root?: Element;
}

type LinkRules = Pick<LinkForesightOptions, "router" | "isAppPage" | "namesRecord">;

/**
 * Whether a link leads to a page of this app that is worth starting ahead of the click: a
 * route the router draws whose path has no params and whose query names no record (a list, a
 * board, a settings page, Home's shortcuts, the sidebar). A link to a record is not, whether
 * the record is in the path (`/shipments/$id`) or in the query (a list with the record's panel
 * open, `?panelEntityId=ord_…`): a list holds one per row, and watching them all cost more on
 * every pointer move than starting them saved; the router's intent preload starts a record on
 * hover or focus. Neither is another origin, a download, a new tab or window, or a path no
 * route matches.
 */
export function isPageLink(
  anchor: HTMLAnchorElement,
  { router, isAppPage, namesRecord }: LinkRules,
): boolean {
  if (!anchor.hasAttribute("href") || anchor.hasAttribute("download")) return false;
  if (anchor.target !== "" && anchor.target !== "_self") return false;
  if (anchor.origin !== window.location.origin) return false;
  const url = new URL(anchor.href);
  if (!isAppPage(url) || namesRecord(url)) return false;

  const [, params, found] = router.getMatchedRoutes(anchor.pathname);
  return found !== undefined && Object.keys(params).length === 0;
}

function isCurrentLocation(anchor: HTMLAnchorElement): boolean {
  return (
    anchor.pathname === window.location.pathname && anchor.search === window.location.search
  );
}

function anchorsIn(node: Node): HTMLAnchorElement[] {
  if (!(node instanceof Element)) return [];
  const found: HTMLAnchorElement[] = [];
  if (node instanceof HTMLAnchorElement) found.push(node);
  for (const element of node.querySelectorAll("a[href]")) {
    if (element instanceof HTMLAnchorElement) found.push(element);
  }
  return found;
}

/**
 * Starts a page before the person reaches its link. Every link to a page of the app
 * (`isPageLink`) is handed to ForesightJS, which predicts it from the pointer's path and from
 * a touch, and the prediction preloads the route: its code, then its loader, so the page's
 * `prefetch` warms the data it opens with. A link stays registered and may warm again after
 * `PAGE_REWARM_MS`; a query still fresh is not fetched again.
 *
 * Changes to the document are gathered and looked at when the browser is idle, never inside
 * the render that made them. Links are released as they leave the document, since
 * ForesightJS keeps a detached element registered.
 *
 * Returns the function that stops watching.
 */
export function watchLinks({
  router,
  isAppPage,
  namesRecord,
  root = document.body,
}: LinkForesightOptions): () => void {
  const rules: LinkRules = { router, isAppPage, namesRecord };
  const manager = ForesightManager.initialize({
    // Scrolling would predict every link in the direction of travel.
    enableScrollPrediction: false,
    // Keyboard focus already preloads through the router, and the tab order is rebuilt after
    // every change to the document.
    enableTabPrediction: false,
    setDataAttributes: false,
  });

  const tracked = new Set<HTMLAnchorElement>();
  const added = new Set<Node>();
  const removed = new Set<Node>();
  const changed = new Set<HTMLAnchorElement>();
  let scanScheduled = false;
  let stopped = false;

  const release = (anchor: HTMLAnchorElement) => {
    if (tracked.delete(anchor)) manager.unregister(anchor);
  };

  const startPage = (anchor: HTMLAnchorElement) => {
    if (!isPageLink(anchor, rules)) {
      release(anchor);
      return undefined;
    }
    if (isCurrentLocation(anchor)) return undefined;
    return router.preloadRoute({
      to: anchor.pathname,
      search: router.options.parseSearch(anchor.search),
      hash: anchor.hash.replace(/^#/, ""),
    });
  };

  const consider = (anchor: HTMLAnchorElement) => {
    if (!isPageLink(anchor, rules)) {
      release(anchor);
      return;
    }
    if (tracked.has(anchor)) return;
    tracked.add(anchor);
    manager.register({
      element: anchor,
      name: anchor.pathname,
      callback: () => startPage(anchor),
      reactivateAfter: PAGE_REWARM_MS,
    });
  };

  const scan = () => {
    scanScheduled = false;
    if (stopped) return;
    if (tracked.size > 0) {
      for (const node of removed) {
        for (const anchor of anchorsIn(node)) {
          if (!anchor.isConnected) release(anchor);
        }
      }
    }
    for (const node of added) {
      if (!node.isConnected) continue;
      for (const anchor of anchorsIn(node)) consider(anchor);
    }
    for (const anchor of changed) {
      if (anchor.isConnected) consider(anchor);
    }
    removed.clear();
    added.clear();
    changed.clear();
  };

  const observer = new MutationObserver((records) => {
    for (const record of records) {
      if (record.type === "attributes") {
        if (record.target instanceof HTMLAnchorElement) changed.add(record.target);
        continue;
      }
      for (const node of record.removedNodes) removed.add(node);
      for (const node of record.addedNodes) added.add(node);
    }
    if (!scanScheduled) {
      scanScheduled = true;
      whenBrowserIdle(scan, LINK_SCAN_IDLE_TIMEOUT_MS);
    }
  });

  for (const anchor of anchorsIn(root)) consider(anchor);
  observer.observe(root, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["href", "target", "download"],
  });

  return () => {
    stopped = true;
    observer.disconnect();
    for (const anchor of tracked) manager.unregister(anchor);
    tracked.clear();
  };
}
