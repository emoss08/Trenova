import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const post = vi.fn();
const get = vi.fn();

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const original = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return { ...original, api: { post, get, delete: vi.fn() } };
});

const { ApiRequestError } = await import("@trenova/shared/lib/api");
const { enablePush } = await import("../push");

function fakeSubscription(endpoint: string) {
  return {
    endpoint,
    toJSON: () => ({ keys: { p256dh: `${endpoint}-p256dh`, auth: `${endpoint}-auth` } }),
    unsubscribe: vi.fn().mockResolvedValue(true),
  };
}

describe("enablePush", () => {
  const existing = fakeSubscription("https://fcm.googleapis.com/fcm/send/shared-device");
  const fresh = fakeSubscription("https://fcm.googleapis.com/fcm/send/fresh");
  const pushManager = {
    getSubscription: vi.fn(),
    subscribe: vi.fn(),
  };

  beforeEach(() => {
    post.mockReset();
    get.mockReset().mockResolvedValue({
      enabled: true,
      publicKey: "BEl62iUYgUivxIkv69yViEuiBIa",
    });
    existing.unsubscribe.mockClear();
    pushManager.getSubscription.mockReset().mockResolvedValue(existing);
    pushManager.subscribe.mockReset().mockResolvedValue(fresh);
    vi.stubGlobal("PushManager", function PushManager() {});
    vi.stubGlobal("Notification", { requestPermission: vi.fn().mockResolvedValue("granted") });
    Object.defineProperty(navigator, "serviceWorker", {
      configurable: true,
      value: {
        register: vi.fn().mockResolvedValue({ pushManager }),
        ready: Promise.resolve(),
      },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("registers the browser's existing subscription", async () => {
    post.mockResolvedValue({});

    await enablePush();

    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("/push/subscriptions/", {
      endpoint: existing.endpoint,
      p256dh: `${existing.endpoint}-p256dh`,
      auth: `${existing.endpoint}-auth`,
    });
    expect(existing.unsubscribe).not.toHaveBeenCalled();
  });

  it("replaces a subscription another account holds instead of taking it over", async () => {
    post
      .mockRejectedValueOnce(new ApiRequestError(409, { title: "Conflict", status: 409 }))
      .mockResolvedValueOnce({});

    await enablePush();

    expect(existing.unsubscribe).toHaveBeenCalledTimes(1);
    expect(pushManager.subscribe).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenLastCalledWith("/push/subscriptions/", {
      endpoint: fresh.endpoint,
      p256dh: `${fresh.endpoint}-p256dh`,
      auth: `${fresh.endpoint}-auth`,
    });
  });

  it("surfaces any other failure without touching the subscription", async () => {
    const failure = new ApiRequestError(422, { title: "Invalid", status: 422 });
    post.mockRejectedValueOnce(failure);

    await expect(enablePush()).rejects.toBe(failure);
    expect(existing.unsubscribe).not.toHaveBeenCalled();
    expect(post).toHaveBeenCalledTimes(1);
  });
});
