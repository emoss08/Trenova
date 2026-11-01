import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const register = vi.fn();

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const original = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return {
    ...original,
    requestGraphQL: vi.fn(
      async ({ operationName, variables }: { operationName: string; variables?: unknown }) => {
        if (operationName === "WebPushConfig") {
          return { webPushConfig: { enabled: true, publicKey: "BEl62iUYgUivxIkv69yViEuiBIa" } };
        }
        if (operationName === "RegisterMyPushSubscription") {
          return register(variables);
        }
        throw new Error(`unexpected operation ${operationName}`);
      },
    ),
  };
});

const { GraphQLRequestError } = await import("@trenova/shared/lib/graphql");
const { enablePush } = await import("../push");

function graphQLFailure(status: number, type: string) {
  return new GraphQLRequestError({
    kind: "graphql",
    message: "Request failed",
    status,
    graphQLErrors: [{ message: "Request failed", extensions: { type }, type }],
  });
}

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
    register.mockReset();
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
    register.mockResolvedValue({ registerMyPushSubscription: { id: "ups_1" } });

    await enablePush();

    expect(register).toHaveBeenCalledTimes(1);
    expect(register).toHaveBeenCalledWith({
      input: {
        endpoint: existing.endpoint,
        p256dh: `${existing.endpoint}-p256dh`,
        auth: `${existing.endpoint}-auth`,
      },
    });
    expect(existing.unsubscribe).not.toHaveBeenCalled();
  });

  it("replaces a subscription another account holds instead of taking it over", async () => {
    register
      .mockRejectedValueOnce(graphQLFailure(409, "https://example.com/problems/conflict"))
      .mockResolvedValueOnce({ registerMyPushSubscription: { id: "ups_2" } });

    await enablePush();

    expect(existing.unsubscribe).toHaveBeenCalledTimes(1);
    expect(pushManager.subscribe).toHaveBeenCalledTimes(1);
    expect(register).toHaveBeenLastCalledWith({
      input: {
        endpoint: fresh.endpoint,
        p256dh: `${fresh.endpoint}-p256dh`,
        auth: `${fresh.endpoint}-auth`,
      },
    });
  });

  it("surfaces any other failure without touching the subscription", async () => {
    const failure = graphQLFailure(422, "https://example.com/problems/validation-error");
    register.mockRejectedValueOnce(failure);

    await expect(enablePush()).rejects.toBe(failure);
    expect(existing.unsubscribe).not.toHaveBeenCalled();
    expect(register).toHaveBeenCalledTimes(1);
  });
});
