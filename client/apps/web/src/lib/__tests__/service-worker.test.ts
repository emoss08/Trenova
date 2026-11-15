import { createTestRouter } from "@trenova/shared/testing/test-router";
import { describe, expect, it, vi } from "vitest";
import { releaseCheck, reloadIntoNewRelease, settleOnNewestRelease } from "../service-worker";

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

async function setup({
  controlled = true,
  safe = () => true,
}: { controlled?: boolean; safe?: () => boolean } = {}) {
  const router = createTestRouter(undefined, ["/shipments"]);
  await router.load();
  const worker = fakeContainer(controlled);
  const assign = vi.fn();
  reloadIntoNewRelease({
    router,
    container: worker.container as never,
    reloadIsSafe: safe,
    target: { location: { assign } },
  });
  const navigate = (pathname: string, search = "") =>
    router.navigate({ href: `${pathname}${search}` });
  return { navigate, ...worker, assign };
}

describe("reloadIntoNewRelease", () => {
  it("opens the next page by a full page load once a new release has taken over", async () => {
    const { navigate, newWorkerTakesOver, assign } = await setup();

    newWorkerTakesOver();
    await navigate("/customers", "?status=active");

    expect(assign).toHaveBeenCalledTimes(1);
    expect(assign).toHaveBeenCalledWith("/customers?status=active");
  });

  it("changes page in place while no new release has arrived", async () => {
    const { navigate, assign } = await setup();

    await navigate("/customers");

    expect(assign).not.toHaveBeenCalled();
  });

  it("does not count the worker's first install as a release", async () => {
    const { navigate, newWorkerTakesOver, assign } = await setup({ controlled: false });

    newWorkerTakesOver();
    await navigate("/customers");

    expect(assign).not.toHaveBeenCalled();
  });

  it("never reloads for a filter, sort or table page, only for a change of page", async () => {
    const { navigate, newWorkerTakesOver, assign } = await setup();

    newWorkerTakesOver();
    await navigate("/shipments", "?status=in-transit");
    await navigate("/shipments", "?status=in-transit&page=2");
    expect(assign).not.toHaveBeenCalled();

    await navigate("/customers");
    expect(assign).toHaveBeenCalledWith("/customers");
  });

  it("waits while a guard holds the page (an unsaved form, a popout), then reloads at a later change of page", async () => {
    let popoutOpen = true;
    const { navigate, newWorkerTakesOver, assign } = await setup({ safe: () => !popoutOpen });

    newWorkerTakesOver();
    await navigate("/customers");
    expect(assign).not.toHaveBeenCalled();

    popoutOpen = false;
    await navigate("/workers");
    expect(assign).toHaveBeenCalledWith("/workers");
  });

  it("tries again at the next change of page when the browser stayed after all", async () => {
    const { navigate, newWorkerTakesOver, assign } = await setup();

    newWorkerTakesOver();
    await navigate("/customers");
    await navigate("/workers");

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
