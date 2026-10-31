import { afterEach, describe, expect, it, vi } from "vitest";
import { addReloadGuard, reloadIsSafe } from "../reload-guards";

const removers: Array<() => void> = [];
afterEach(() => {
  while (removers.length) removers.pop()?.();
});

describe("reload guards", () => {
  it("allows a reload when nothing holds work", () => {
    expect(reloadIsSafe()).toBe(true);
  });

  it("holds while any guard holds work, and asks each guard at the moment of the reload", () => {
    let popoutOpen = false;
    removers.push(addReloadGuard(() => popoutOpen));
    removers.push(addReloadGuard(() => false));
    expect(reloadIsSafe()).toBe(true);

    popoutOpen = true;
    expect(reloadIsSafe()).toBe(false);
  });

  it("stops holding once the guard is removed", () => {
    const remove = addReloadGuard(() => true);
    expect(reloadIsSafe()).toBe(false);

    remove();
    expect(reloadIsSafe()).toBe(true);
  });
});

describe("leaveRegardlessOfWork", () => {
  function stubLocation() {
    const location = { ...window.location, assign: vi.fn() };
    Object.defineProperty(window, "location", { configurable: true, value: location });
    return location;
  }

  async function freshModule() {
    vi.resetModules();
    return import("../reload-guards");
  }

  it("is not leaving until asked to", async () => {
    const { isLeavingRegardlessOfWork } = await freshModule();

    expect(isLeavingRegardlessOfWork()).toBe(false);
  });

  it("marks the page as leaving before it loads the destination", async () => {
    const { isLeavingRegardlessOfWork, leaveRegardlessOfWork } = await freshModule();
    const location = stubLocation();
    location.assign.mockImplementation(() => {
      expect(isLeavingRegardlessOfWork()).toBe(true);
    });

    leaveRegardlessOfWork("/login?signed_out=elsewhere");

    expect(location.assign).toHaveBeenCalledWith("/login?signed_out=elsewhere");
  });
});
