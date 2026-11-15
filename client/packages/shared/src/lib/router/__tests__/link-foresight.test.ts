import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PAGE_REWARM_MS, isPageLink, watchLinks, type PredictingRouter } from "../link-foresight";

const foresight = vi.hoisted(() => ({
  register: vi.fn(),
  unregister: vi.fn(),
  initialize: vi.fn(),
}));

const idle = vi.hoisted(() => ({ tasks: [] as Array<() => void> }));

vi.mock("js.foresight", () => ({
  ForesightManager: {
    initialize: (settings: unknown) => {
      foresight.initialize(settings);
      return { register: foresight.register, unregister: foresight.unregister };
    },
  },
}));

vi.mock("@trenova/shared/lib/when-page-settles", () => ({
  whenBrowserIdle: (task: () => void) => {
    idle.tasks.push(task);
  },
}));

const ROUTES: Record<string, Record<string, string>> = {
  "/": {},
  "/shipments": {},
  "/customers": {},
  "/shipments/shp_1": { id: "shp_1" },
};

function fakeRouter() {
  const preloadRoute = vi.fn(
    (_options: { to: string; search: Record<string, unknown>; hash: string }) =>
      Promise.resolve(),
  );
  const router: PredictingRouter = {
    options: {
      parseSearch: (searchStr) => Object.fromEntries(new URLSearchParams(searchStr)),
    },
    getMatchedRoutes: (pathname) => {
      const params = ROUTES[pathname];
      if (!params) return [[{ id: "__root__" }], {}, undefined];
      return [[{ id: "__root__" }, { id: pathname }], params, { id: pathname }];
    },
    preloadRoute,
  };
  return { router, preloadRoute };
}

const isAppPage = (url: URL) => !url.pathname.startsWith("/api/");
const namesRecord = (url: URL) => /=[a-z]+_[0-9A-Z]{26}/.test(url.search);
const rules = (router: PredictingRouter) => ({ router, isAppPage, namesRecord });

function link(href: string, attributes: Record<string, string> = {}): HTMLAnchorElement {
  const anchor = document.createElement("a");
  anchor.setAttribute("href", href);
  for (const [name, value] of Object.entries(attributes)) anchor.setAttribute(name, value);
  return anchor;
}

type Registration = {
  element: Element;
  callback: () => Promise<unknown> | undefined;
  reactivateAfter: number;
};

function registrations(): Registration[] {
  return foresight.register.mock.calls.map(([options]) => options as Registration);
}

function registeredElements(): Element[] {
  return registrations().map((options) => options.element);
}

function registration(anchor: HTMLAnchorElement): Registration {
  const found = registrations().find((options) => options.element === anchor);
  if (!found) throw new Error(`${anchor.getAttribute("href")} was not registered`);
  return found;
}

/** Lets the observer deliver its records, then runs the idle-time scan they scheduled. */
async function documentSettles(): Promise<void> {
  await new Promise<void>((resolve) => setTimeout(resolve, 0));
  const tasks = idle.tasks.splice(0);
  for (const task of tasks) task();
}

describe("isPageLink", () => {
  const { router } = fakeRouter();

  it("accepts a link to a route without path params", () => {
    expect(isPageLink(link("/shipments?status=late"), rules(router))).toBe(true);
  });

  it("accepts a link that names this window as its target", () => {
    expect(isPageLink(link("/customers", { target: "_self" }), rules(router))).toBe(true);
  });

  it.each([
    ["a record in the path", link("/shipments/shp_1")],
    [
      "a record in the query",
      link("/shipments?panelType=edit&panelEntityId=shp_01M5N1C3S81WFBMWEZ1EM840HF"),
    ],
    ["another origin", link("https://example.com/shipments")],
    ["a download", link("/shipments", { download: "" })],
    ["a new tab", link("/shipments", { target: "_blank" })],
    ["a path that is not a page of the app", link("/api/shipments")],
    ["a path no route matches", link("/nowhere")],
    ["an anchor without an href", document.createElement("a")],
  ])("rejects %s", (_case, anchor) => {
    expect(isPageLink(anchor, rules(router))).toBe(false);
  });
});

describe("watchLinks", () => {
  let root: HTMLDivElement;
  let stop: () => void = () => {};

  beforeEach(() => {
    foresight.register.mockClear();
    foresight.unregister.mockClear();
    foresight.initialize.mockClear();
    idle.tasks.length = 0;
    window.history.replaceState(null, "", "/");
    root = document.createElement("div");
    document.body.append(root);
  });

  afterEach(() => {
    stop();
    root.remove();
  });

  it("predicts from the pointer and touch only", () => {
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    expect(foresight.initialize).toHaveBeenCalledWith({
      enableScrollPrediction: false,
      enableTabPrediction: false,
      setDataAttributes: false,
    });
  });

  it("registers the page links already in the document and nothing else", () => {
    const shipments = link("/shipments");
    root.append(
      shipments,
      link("/shipments/shp_1"),
      link("/api/files/1"),
      link("https://example.com/"),
    );

    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    expect(registeredElements()).toEqual([shipments]);
    expect(registration(shipments).reactivateAfter).toBe(PAGE_REWARM_MS);
  });

  it("preloads the route with the link's path, query and hash when predicted", async () => {
    const { router, preloadRoute } = fakeRouter();
    const late = link("/shipments?status=late&sort=eta#board");
    root.append(late);
    stop = watchLinks({ router, isAppPage, namesRecord, root });

    await registration(late).callback();

    expect(preloadRoute.mock.calls).toEqual([
      [{ to: "/shipments", search: { status: "late", sort: "eta" }, hash: "board" }],
    ]);
    expect(foresight.unregister).not.toHaveBeenCalled();
  });

  it("preloads nothing for a link to where the person already is", async () => {
    const { router, preloadRoute } = fakeRouter();
    window.history.replaceState(null, "", "/shipments?status=late");
    const here = link("/shipments?status=late");
    root.append(here);
    stop = watchLinks({ router, isAppPage, namesRecord, root });

    await registration(here).callback();

    expect(preloadRoute).not.toHaveBeenCalled();
  });

  it("looks at links entering the document only once the browser is idle", async () => {
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });
    const row = document.createElement("div");
    const customers = link("/customers");
    row.append(customers);

    root.append(row);
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    expect(registeredElements()).toEqual([]);

    await documentSettles();
    expect(registeredElements()).toEqual([customers]);
  });

  it("schedules one scan for a burst of changes", async () => {
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    root.append(link("/customers"));
    root.append(link("/shipments"));
    await new Promise<void>((resolve) => setTimeout(resolve, 0));

    expect(idle.tasks).toHaveLength(1);
  });

  it("ignores a link that entered and left before the scan", async () => {
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });
    const brief = link("/customers");

    root.append(brief);
    brief.remove();
    await documentSettles();

    expect(registeredElements()).toEqual([]);
  });

  it("releases a link when it leaves the document", async () => {
    const shipments = link("/shipments");
    root.append(shipments);
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    shipments.remove();
    await documentSettles();

    expect(foresight.unregister).toHaveBeenCalledWith(shipments);
  });

  it("keeps a link that was moved within the document", async () => {
    const shipments = link("/shipments");
    const elsewhere = document.createElement("div");
    root.append(shipments, elsewhere);
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    elsewhere.append(shipments);
    await documentSettles();

    expect(foresight.unregister).not.toHaveBeenCalled();
    expect(registeredElements()).toEqual([shipments]);
  });

  it("releases a link whose href changes to a record", async () => {
    const shipments = link("/shipments");
    root.append(shipments);
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    shipments.setAttribute("href", "/shipments/shp_1");
    await documentSettles();

    expect(foresight.unregister).toHaveBeenCalledWith(shipments);
  });

  it("registers a link whose href changes from a record to a page", async () => {
    const anchor = link("/shipments/shp_1");
    root.append(anchor);
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    anchor.setAttribute("href", "/customers");
    await documentSettles();

    expect(registeredElements()).toEqual([anchor]);
  });

  it("releases every link it holds when stopped, and scans nothing after", async () => {
    const shipments = link("/shipments");
    const customers = link("/customers");
    root.append(shipments, customers);
    stop = watchLinks({ router: fakeRouter().router, isAppPage, namesRecord, root });

    root.append(link("/"));
    await new Promise<void>((resolve) => setTimeout(resolve, 0));
    stop();
    stop = () => {};
    await documentSettles();

    expect(foresight.unregister.mock.calls.map(([element]) => element)).toEqual([
      shipments,
      customers,
    ]);
    expect(registeredElements()).toEqual([shipments, customers]);
  });
});
