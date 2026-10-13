import { afterEach, describe, expect, it } from "vitest";
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
