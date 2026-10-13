import { describe, expect, it, vi } from "vitest";
import type { Location, RouterState } from "react-router";
import { releaseCheck, reloadIntoNewRelease, settleOnNewestRelease } from "../service-worker";

type Listener = (state: RouterState) => void;

function location(pathname: string, search = ""): Location {
  return { pathname, search, hash: "", state: null, key: pathname + search };
}

/** A router whose navigations the test drives, the way React Router reports them. */
function fakeRouter(start: string) {
  const listeners: Listener[] = [];
  let state = {
    location: location(start),
    navigation: { state: "idle", location: undefined },
  } as unknown as RouterState;
  const emit = (next: Partial<RouterState>) => {
    state = { ...state, ...next } as RouterState;
    for (const listener of listeners) listener(state);
  };
  return {
    router: {
      get state() {
        return state;
      },
      subscribe: (listener: Listener) => {
        listeners.push(listener);
        return () => void listeners.splice(listeners.indexOf(listener), 1);
      },
      createHref: (to: Location | URL) =>
        to instanceof URL ? to.href : `${to.pathname}${to.search}${to.hash}`,
    },
    /** A navigation with a loading phase (a route with a loader or lazy code). */
    navigate: (pathname: string, search = "") => {
      const to = location(pathname, search);
      emit({ navigation: { state: "loading", location: to } as RouterState["navigation"] });
      emit({
        location: to,
        navigation: { state: "idle", location: undefined } as RouterState["navigation"],
      });
    },
    /** A navigation that commits at once, with no loading phase. */
    navigateInstantly: (pathname: string, search = "") =>
      emit({
        location: location(pathname, search),
        navigation: { state: "idle", location: undefined } as RouterState["navigation"],
      }),
  };
}

function fakeContainer(controlled: boolean) {
  const events = new EventTarget();
  return {
    container: {
      controller: (controlled ? {} : null) as ServiceWorker | null,
      addEventListener: events.addEventListener.bind(events),
    },
    newWorkerTakesOver: () => events.dispatchEvent(new Event("controllerchange")),
  };
}

function setup({
  controlled = true,
  safe = () => true,
}: { controlled?: boolean; safe?: () => boolean } = {}) {
  const routing = fakeRouter("/shipments");
  const worker = fakeContainer(controlled);
  const assign = vi.fn();
  reloadIntoNewRelease({
    router: routing.router as never,
    container: worker.container as never,
    reloadIsSafe: safe,
    target: { location: { assign } },
  });
  return { ...routing, ...worker, assign };
}

describe("reloadIntoNewRelease", () => {
  it("opens the next page by a full page load once a new release has taken over", () => {
    const { navigate, newWorkerTakesOver, assign } = setup();

    newWorkerTakesOver();
    navigate("/customers", "?status=active");

    expect(assign).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith("/customers?status=active");
  });

  it("changes page in place while no new release has arrived", () => {
    const { navigate, assign } = setup();

    navigate("/customers");

    expect(assign).not.toHaveBeenCalled();
  });

  it("does not count the worker's first install as a release", () => {
    const { navigate, newWorkerTakesOver, assign } = setup({ controlled: false });

    newWorkerTakesOver();
    navigate("/customers");

    expect(assign).not.toHaveBeenCalled();
  });

  it("never reloads for a filter, sort or table page, only for a change of page", () => {
    const { navigateInstantly, navigate, newWorkerTakesOver, assign } = setup();

    newWorkerTakesOver();
    navigateInstantly("/shipments", "?status=in-transit");
    navigateInstantly("/shipments", "?status=in-transit&page=2");
    expect(assign).not.toHaveBeenCalled();

    navigate("/customers");
    expect(assign).toHaveBeenCalledWith("/customers");
  });

  it("reloads after a change of page that commits without a loading phase", () => {
    const { navigateInstantly, newWorkerTakesOver, assign } = setup();

    newWorkerTakesOver();
    navigateInstantly("/customers");

    expect(assign).toHaveBeenCalledWith("/customers");
  });

  it("waits while a guard holds the page (an unsaved form, a popout), then reloads at a later change of page", () => {
    let popoutOpen = true;
    const { navigate, newWorkerTakesOver, assign } = setup({ safe: () => !popoutOpen });

    newWorkerTakesOver();
    navigate("/customers");
    expect(assign).not.toHaveBeenCalled();

    popoutOpen = false;
    navigate("/workers");
    expect(assign).toHaveBeenCalledWith("/workers");
  });

  it("tries again at the next change of page when the browser stayed after all", () => {
    const { navigate, newWorkerTakesOver, assign } = setup();

    newWorkerTakesOver();
    navigate("/customers");
    navigate("/workers");

    expect(assign.mock.calls).toEqual([["/customers"], ["/workers"]]);
  });
});

describe("releaseCheck", () => {
  it("asks for a new release at most once a minute", () => {
    let now = 0;
    const update = vi.fn().mockResolvedValue(undefined);
    const check = releaseCheck({ update }, () => now);

    check();
    now = 30_000;
    check();
    now = 61_000;
    check();

    expect(update).toHaveBeenCalledTimes(2);
  });

  it("survives a failed check", async () => {
    const update = vi.fn().mockRejectedValue(new Error("offline"));
    const check = releaseCheck({ update }, () => 0);

    expect(() => check()).not.toThrow();
    await Promise.resolve();
    expect(update).toHaveBeenCalledTimes(1);
  });
});

describe("settleOnNewestRelease", () => {
  it("resolves at once when there is no worker", async () => {
    await expect(settleOnNewestRelease(undefined, 5000)).resolves.toBeUndefined();
    await expect(
      settleOnNewestRelease({ getRegistration: async () => undefined }, 5000),
    ).resolves.toBeUndefined();
  });

  it("waits for a newly found release's worker to take over", async () => {
    const incoming = Object.assign(new EventTarget(), {
      state: "installing" as ServiceWorkerState,
    });
    const registration = {
      update: vi.fn().mockResolvedValue(undefined),
      installing: incoming,
      waiting: null,
    };
    let settled = false;
    const done = settleOnNewestRelease(
      { getRegistration: async () => registration as never },
      5000,
    ).then(() => (settled = true));

    await vi.waitFor(() => expect(registration.update).toHaveBeenCalled());
    expect(settled).toBe(false);

    incoming.state = "activated";
    incoming.dispatchEvent(new Event("statechange"));
    await done;
    expect(settled).toBe(true);
  });

  it("gives up waiting after the time allowed", async () => {
    const incoming = Object.assign(new EventTarget(), {
      state: "installing" as ServiceWorkerState,
    });
    const registration = {
      update: vi.fn().mockResolvedValue(undefined),
      installing: incoming,
      waiting: null,
    };

    await expect(
      settleOnNewestRelease({ getRegistration: async () => registration as never }, 20),
    ).resolves.toBeUndefined();
  });

  it("never rejects, even when the check fails", async () => {
    const registration = {
      update: vi.fn().mockRejectedValue(new Error("offline")),
      installing: null,
      waiting: null,
    };
    await expect(
      settleOnNewestRelease({ getRegistration: async () => registration as never }, 20),
    ).resolves.toBeUndefined();
  });
});
