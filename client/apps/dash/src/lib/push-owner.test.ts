import { beforeEach, describe, expect, it, vi } from "vitest";
import { dropPushOfAnotherDriver, rememberPushOwner, releasePush } from "./push-owner";

const mocks = vi.hoisted(() => ({
  disablePush: vi.fn(),
  getPushSubscription: vi.fn(),
}));

vi.mock("@trenova/shared/lib/push", () => mocks);

describe("push owner", () => {
  beforeEach(() => {
    window.localStorage.clear();
    mocks.disablePush.mockReset().mockResolvedValue(undefined);
    mocks.getPushSubscription.mockReset().mockResolvedValue({ endpoint: "https://push.test/a" });
  });

  it("keeps the alerts of the driver who owns them", async () => {
    rememberPushOwner("usr_a");

    await dropPushOfAnotherDriver("usr_a");

    expect(mocks.disablePush).not.toHaveBeenCalled();
  });

  it("stops the alerts when another driver signs in on the browser", async () => {
    rememberPushOwner("usr_a");

    await dropPushOfAnotherDriver("usr_b");

    expect(mocks.disablePush).toHaveBeenCalledTimes(1);
  });

  it("has nothing to stop when the browser holds no subscription", async () => {
    mocks.getPushSubscription.mockResolvedValue(null);

    await dropPushOfAnotherDriver("usr_b");

    expect(mocks.disablePush).not.toHaveBeenCalled();
  });

  it("forgets the owner when the driver signs out", async () => {
    rememberPushOwner("usr_a");

    await releasePush();

    expect(window.localStorage.getItem("trenova-dash-push-owner")).toBeNull();
    expect(mocks.disablePush).toHaveBeenCalledTimes(1);
  });
});
