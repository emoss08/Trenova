import { describe, expect, it, vi } from "vitest";
import { installChunkRecovery } from "../chunk-recovery";

type PageOptions = {
  running?: () => string;
  deployed?: () => Promise<string | null>;
  beforeReload?: () => Promise<void>;
  canReload?: () => boolean;
  storage?: () => Pick<Storage, "getItem" | "setItem">;
};

function page({
  running = () => "/assets/js/main.old.js",
  deployed = async () => "/assets/js/main.new.js",
  beforeReload = async () => undefined,
  canReload = () => true,
  storage,
}: PageOptions = {}) {
  const target = Object.assign(new EventTarget(), { location: { reload: vi.fn() } });
  const entries = new Map<string, string>();
  installChunkRecovery({
    target: target as unknown as Window,
    storage:
      storage ??
      (() => ({
        getItem: (key: string) => entries.get(key) ?? null,
        setItem: (key: string, value: string) => void entries.set(key, value),
      })),
    currentBuild: running,
    deployedBuild: deployed,
    beforeReload,
    canReload,
  });
  const failChunk = () => {
    const event = new Event("vite:preloadError", { cancelable: true });
    target.dispatchEvent(event);
    return event;
  };
  const settled = () => new Promise((resolve) => setTimeout(resolve, 20));
  return { reload: target.location.reload, failChunk, settled };
}

describe("installChunkRecovery", () => {
  it("reloads into the new release once the server confirms one is deployed", async () => {
    const { reload, failChunk } = page();

    failChunk();

    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
  });

  it("lets the failure show as it would anyway, rather than swallowing it", () => {
    const { failChunk } = page();

    expect(failChunk().defaultPrevented).toBe(false);
  });

  it("does not reload when the same build is still deployed", async () => {
    const { reload, failChunk, settled } = page({ deployed: async () => "/assets/js/main.old.js" });

    failChunk();
    await settled();

    expect(reload).not.toHaveBeenCalled();
  });

  it("does not reload when the server cannot be asked, as when offline", async () => {
    const { reload, failChunk, settled } = page({ deployed: async () => null });

    failChunk();
    await settled();

    expect(reload).not.toHaveBeenCalled();
  });

  it("lets the new release's worker take over before reloading", async () => {
    const order: string[] = [];
    let activate: () => void = () => undefined;
    const { reload, failChunk } = page({
      beforeReload: () =>
        new Promise<void>((resolve) => {
          order.push("waiting for the new release");
          activate = () => {
            order.push("new release active");
            resolve();
          };
        }),
    });
    reload.mockImplementation(() => void order.push("reload"));

    failChunk();
    await vi.waitFor(() => expect(order).toEqual(["waiting for the new release"]));
    activate();

    await vi.waitFor(() =>
      expect(order).toEqual(["waiting for the new release", "new release active", "reload"]),
    );
  });

  it("reloads even when waiting for the new release fails", async () => {
    const { reload, failChunk } = page({
      beforeReload: () => Promise.reject(new Error("offline")),
    });

    failChunk();

    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
  });

  it("asks the server once for a burst of failures", async () => {
    const deployed = vi.fn(async () => "/assets/js/main.new.js");
    const { reload, failChunk } = page({ deployed });

    failChunk();
    failChunk();
    failChunk();

    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
    expect(deployed).toHaveBeenCalledTimes(1);
  });

  it("reloads only once per build, so a release that keeps failing cannot loop", async () => {
    const { reload, failChunk, settled } = page();

    failChunk();
    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
    failChunk();
    await settled();

    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("leaves the page alone while it holds work, and keeps its one reload for later", async () => {
    let formDirty = true;
    const { reload, failChunk, settled } = page({ canReload: () => !formDirty });

    failChunk();
    await settled();
    expect(reload).not.toHaveBeenCalled();

    formDirty = false;
    failChunk();
    await vi.waitFor(() => expect(reload).toHaveBeenCalledTimes(1));
  });

  it("does not reload when it cannot remember having reloaded", async () => {
    const { reload, failChunk, settled } = page({
      storage: () => {
        throw new DOMException("denied", "SecurityError");
      },
    });

    failChunk();
    await settled();

    expect(reload).not.toHaveBeenCalled();
  });
});
