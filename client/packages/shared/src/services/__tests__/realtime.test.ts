import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const apiPost = vi.fn();
const apiDelete = vi.fn();

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    post: (...args: unknown[]) => apiPost(...args),
    delete: (...args: unknown[]) => apiDelete(...args),
  },
}));

const { RealtimeClient, REALTIME_USERS_SCOPE } = await import("@trenova/shared/services/realtime");

/**
 * One response from GET /realtime/stream/, written the way the Go handler
 * writes it: an optional id line, an event line, one data line, a blank line.
 */
class FakeStream {
  private controller: ReadableStreamDefaultController<Uint8Array> | null = null;
  private readonly encoder = new TextEncoder();
  readonly body: ReadableStream<Uint8Array>;

  constructor() {
    this.body = new ReadableStream<Uint8Array>({
      start: (controller) => {
        this.controller = controller;
      },
    });
  }

  send(event: string, data: unknown, id?: string) {
    const idLine = id ? `id: ${id}\n` : "";
    this.controller?.enqueue(
      this.encoder.encode(`${idLine}event: ${event}\ndata: ${JSON.stringify(data)}\n\n`),
    );
  }

  comment() {
    this.controller?.enqueue(this.encoder.encode(": keepalive\n\n"));
  }

  end() {
    try {
      this.controller?.close();
    } catch {
      // Already closed by an abort.
    }
  }
}

type FetchCall = { url: string; init: RequestInit };

function response(status: number, stream?: FakeStream, headers: Record<string, string> = {}) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: new Headers(headers),
    body: stream?.body ?? null,
  } as unknown as Response;
}

let calls: FetchCall[];
let replies: Array<() => Response>;

function header(call: FetchCall, name: string): string | null {
  return new Headers(call.init.headers).get(name);
}

async function tick(ms = 0) {
  await vi.advanceTimersByTimeAsync(ms);
}

function ready(connectionId: string, extra: Record<string, unknown> = {}) {
  return { connectionId, heartbeatIntervalMs: 15_000, resumed: false, ...extra };
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  vi.spyOn(Math, "random").mockReturnValue(0);
  calls = [];
  replies = [];
  apiPost.mockReset();
  apiDelete.mockReset();
  apiDelete.mockResolvedValue(undefined);
  vi.stubGlobal(
    "fetch",
    vi.fn((url: string, init: RequestInit) => {
      calls.push({ url, init });
      const reply = replies.shift();
      if (!reply) {
        return new Promise<Response>(() => undefined);
      }
      return Promise.resolve(reply());
    }),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

function streamReply(): FakeStream {
  const stream = new FakeStream();
  replies.push(() => response(200, stream));
  return stream;
}

describe("RealtimeClient", () => {
  it("opens the stream with the session cookie and joins user presence only when asked", async () => {
    streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: true });
    await tick();

    expect(calls).toHaveLength(1);
    expect(calls[0].url).toMatch(/\/realtime\/stream\/\?presence=users$/);
    expect(calls[0].init.credentials).toBe("include");
    expect(header(calls[0], "Accept")).toBe("text/event-stream");
    expect(header(calls[0], "Last-Event-ID")).toBeNull();
    client.disconnect();

    streamReply();
    const portal = new RealtimeClient();
    portal.connect({ identity: "usr_2", joinUsers: false });
    await tick();
    expect(calls[1].url).toMatch(/\/realtime\/stream\/$/);
    portal.disconnect();
  });

  it("is connected once the server says ready, and delivers invalidations by name", async () => {
    const stream = streamReply();
    const client = new RealtimeClient();
    const states: string[] = [];
    client.onStateChange((state) => states.push(state));
    const received: unknown[] = [];
    client.on("resource.invalidation", (payload) => received.push(payload));

    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    expect(client.getState()).toBe("connecting");

    stream.send("ready", ready("rtc_1"));
    stream.comment();
    stream.send("resource.invalidation", { resource: "shipments", action: "updated" }, "3.10-0");
    await tick();

    expect(states).toEqual(["connecting", "connected"]);
    expect(client.getConnectionId()).toBe("rtc_1");
    expect(received).toEqual([{ resource: "shipments", action: "updated" }]);
    client.disconnect();
  });

  it("resumes from the last event it applied", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();

    first.send("ready", ready("rtc_1", { cursor: "3.1-0" }));
    first.send("resource.invalidation", { resource: "a" }, "3.7-0");
    await tick();
    streamReply();
    first.end();
    await tick(250);

    expect(calls).toHaveLength(2);
    expect(header(calls[1], "Last-Event-ID")).toBe("3.7-0");
    client.disconnect();
  });

  it("resumes from the cursor in ready when no event arrived before the drop", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();

    first.send("ready", ready("rtc_1", { cursor: "5.42-3" }));
    await tick();
    streamReply();
    first.end();
    await tick(250);

    expect(header(calls[1], "Last-Event-ID")).toBe("5.42-3");
    client.disconnect();
  });

  it("keeps its own cursor when the server resumed it", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    first.send("ready", ready("rtc_1", { cursor: "1.1-0" }));
    first.send("resource.invalidation", { resource: "a" }, "1.5-0");
    await tick();

    const second = streamReply();
    first.end();
    await tick(250);
    second.send("ready", ready("rtc_2", { resumed: true, cursor: "1.2-0" }));
    await tick();
    streamReply();
    second.end();
    await tick(250);

    expect(header(calls[2], "Last-Event-ID")).toBe("1.5-0");
    client.disconnect();
  });

  it("reconnects within the rotation spread when the server recycles the stream", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    first.send("ready", ready("rtc_1"));
    first.send("close", { reason: "rotate" });
    await tick();
    streamReply();
    first.end();

    await tick(499);
    expect(calls).toHaveLength(1);
    await tick(1);
    expect(calls).toHaveLength(2);
    client.disconnect();
  });

  it("stops for good when the session is not accepted", async () => {
    replies.push(() => response(401));
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    await tick(120_000);

    expect(calls).toHaveLength(1);
    expect(client.getState()).toBe("disconnected");
  });

  it("waits as long as Retry-After asks when the server is at capacity", async () => {
    replies.push(() => response(429, undefined, { "Retry-After": "7" }));
    streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();

    await tick(7_499);
    expect(calls).toHaveLength(1);
    await tick(1);
    expect(calls).toHaveLength(2);
    client.disconnect();
  });

  it("treats a stream silent past its heartbeat as dead", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    first.send("ready", ready("rtc_1", { heartbeatIntervalMs: 1_000 }));
    await tick();
    streamReply();

    await tick(2_499);
    expect(calls).toHaveLength(1);
    await tick(1);
    await tick(250);
    expect(calls).toHaveLength(2);
    client.disconnect();
  });

  it("builds user presence from the snapshot and each transition", async () => {
    const stream = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: true });
    await tick();
    stream.send("ready", ready("rtc_1"));
    stream.send("presence.snapshot", {
      scope: REALTIME_USERS_SCOPE,
      members: [{ userId: "usr_1", connectionId: "rtc_1", name: "Me" }],
    });
    await tick();

    const seen: string[][] = [];
    client.subscribePresence(REALTIME_USERS_SCOPE, (members) =>
      seen.push(members.map((member) => member.connectionId)),
    );
    expect(seen).toEqual([["rtc_1"]]);

    stream.send("presence", {
      scope: REALTIME_USERS_SCOPE,
      action: "enter",
      userId: "usr_2",
      connectionId: "rtc_2",
      name: "Them",
    });
    stream.send("presence", {
      scope: REALTIME_USERS_SCOPE,
      action: "leave",
      userId: "usr_1",
      connectionId: "rtc_1",
    });
    stream.send("presence", {
      scope: REALTIME_USERS_SCOPE,
      action: "leave",
      userId: "usr_9",
      connectionId: "rtc_unknown",
    });
    await tick();

    expect(seen).toEqual([["rtc_1"], ["rtc_1", "rtc_2"], ["rtc_2"]]);
    client.disconnect();
  });

  it("joins a scope with its connection, rejoins on reconnect, and leaves with the live one", async () => {
    const scope = "shipment-comments:shp_1";
    const path = "/shipments/shp_1/comments/presence/";
    apiPost.mockImplementation((_path: string, body: { connectionId: string }) =>
      Promise.resolve({
        scope,
        members: [{ userId: "usr_2", connectionId: `peer-of-${body.connectionId}`, name: "Pat" }],
      }),
    );

    const client = new RealtimeClient();
    const seen: string[][] = [];
    client.subscribePresence(scope, (members) =>
      seen.push(members.map((member) => member.connectionId)),
    );
    const leave = client.joinScope(scope, path);
    expect(apiPost).not.toHaveBeenCalled();

    const first = streamReply();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    first.send("ready", ready("rtc_1"));
    await tick();

    expect(apiPost).toHaveBeenCalledWith(path, { connectionId: "rtc_1" });
    expect(seen.at(-1)).toEqual(["peer-of-rtc_1"]);

    const second = streamReply();
    first.end();
    await tick(250);
    second.send("ready", ready("rtc_2"));
    await tick();
    expect(apiPost).toHaveBeenLastCalledWith(path, { connectionId: "rtc_2" });

    leave();
    expect(apiDelete).toHaveBeenCalledWith(`${path}?connectionId=rtc_2`);
    expect(seen.at(-1)).toEqual([]);
    client.disconnect();
  });

  it("shares one membership between two joins of the same scope", async () => {
    const scope = "shipment-comments:shp_1";
    const path = "/shipments/shp_1/comments/presence/";
    apiPost.mockResolvedValue({ scope, members: [] });

    const stream = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    stream.send("ready", ready("rtc_1"));
    await tick();

    const leaveViewers = client.joinScope(scope, path);
    const leaveTyping = client.joinScope(scope, path);
    await tick();
    expect(apiPost).toHaveBeenCalledTimes(1);

    leaveViewers();
    expect(apiDelete).not.toHaveBeenCalled();
    leaveTyping();
    leaveTyping();
    expect(apiDelete).toHaveBeenCalledTimes(1);
    client.disconnect();
  });

  it("announces a gap it could not replay as a reset", async () => {
    const stream = streamReply();
    const client = new RealtimeClient();
    const reset = vi.fn();
    client.on("reset", reset);
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    stream.send("ready", ready("rtc_1"));
    stream.send("reset", {});
    await tick();

    expect(reset).toHaveBeenCalledTimes(1);
    client.disconnect();
  });

  it("sends typing only on a live connection, naming that connection", async () => {
    apiPost.mockResolvedValue(undefined);
    const client = new RealtimeClient();
    client.sendTyping("/shipments/shp_1/comments/typing/", false);
    expect(apiPost).not.toHaveBeenCalled();

    const stream = streamReply();
    client.connect({ identity: "usr_1", joinUsers: false });
    await tick();
    stream.send("ready", ready("rtc_7"));
    await tick();

    client.sendTyping("/shipments/shp_1/comments/typing/", true);
    expect(apiPost).toHaveBeenCalledWith("/shipments/shp_1/comments/typing/", {
      connectionId: "rtc_7",
      stop: true,
    });
    client.disconnect();
  });

  it("forgets the previous session's cursor when a different identity connects", async () => {
    const first = streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1:org_1:bu_1", joinUsers: true });
    await tick();
    first.send("ready", ready("rtc_1"));
    first.send("resource.invalidation", { resource: "a" }, "2.9-0");
    await tick();

    streamReply();
    client.connect({ identity: "usr_1:org_2:bu_2", joinUsers: true });
    await tick();

    expect(calls).toHaveLength(2);
    expect(header(calls[1], "Last-Event-ID")).toBeNull();
    client.disconnect();
  });

  it("keeps one stream when asked to connect again with nothing changed", async () => {
    streamReply();
    const client = new RealtimeClient();
    client.connect({ identity: "usr_1", joinUsers: true });
    client.connect({ identity: "usr_1", joinUsers: true });
    await tick();

    expect(calls).toHaveLength(1);
    client.disconnect();
    expect(client.getState()).toBe("disconnected");
  });
});
