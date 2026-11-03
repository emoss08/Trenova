import { describe, expect, it, vi } from "vitest";
import {
  announceUnlessWatched,
  billingTransferResultKey,
  markResultSeen,
  watchResult,
} from "../watched-results";

describe("announceUnlessWatched", () => {
  it("announces a result nothing is showing", () => {
    const announce = vi.fn();

    announceUnlessWatched(billingTransferResultKey("btr_unwatched"), announce);

    expect(announce).toHaveBeenCalledTimes(1);
  });

  it("holds the announcement while a screen is showing the result", () => {
    const announce = vi.fn();
    const release = watchResult(billingTransferResultKey("btr_held"));

    announceUnlessWatched(billingTransferResultKey("btr_held"), announce);

    expect(announce).not.toHaveBeenCalled();
    release();
  });

  // The person saw the result on screen, so the notice has nothing left to tell them.
  it("drops the announcement once the screen has shown the result", () => {
    const announce = vi.fn();
    const key = billingTransferResultKey("btr_seen");
    const release = watchResult(key);

    announceUnlessWatched(key, announce);
    markResultSeen(key);
    release();

    expect(announce).not.toHaveBeenCalled();
  });

  it("drops an announcement that arrives after the result was shown", () => {
    const announce = vi.fn();
    const key = billingTransferResultKey("btr_late");
    const release = watchResult(key);

    markResultSeen(key);
    announceUnlessWatched(key, announce);
    release();

    expect(announce).not.toHaveBeenCalled();
  });

  // Closed before the result appeared: the person never saw it, so the notice still lands.
  it("announces after all when the screen closes before showing the result", () => {
    const announce = vi.fn();
    const key = billingTransferResultKey("btr_closed_early");
    const release = watchResult(key);

    announceUnlessWatched(key, announce);
    release();

    expect(announce).toHaveBeenCalledTimes(1);
  });

  it("waits for the last of several screens showing the same result", () => {
    const announce = vi.fn();
    const key = billingTransferResultKey("btr_two_screens");
    const first = watchResult(key);
    const second = watchResult(key);

    announceUnlessWatched(key, announce);
    first();
    first();
    expect(announce).not.toHaveBeenCalled();

    second();
    expect(announce).toHaveBeenCalledTimes(1);
  });

  it("forgets a result once nothing is watching it", () => {
    const key = billingTransferResultKey("btr_forgotten");
    const release = watchResult(key);
    markResultSeen(key);
    release();

    const announce = vi.fn();
    announceUnlessWatched(key, announce);

    expect(announce).toHaveBeenCalledTimes(1);
  });
});
