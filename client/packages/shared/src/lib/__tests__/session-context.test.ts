import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const CHANNEL_NAME = "trenova-session-context";
const PROBLEM_BASE = "https://trenova.app/problems/";

const LOADED = { userId: "usr_loaded", organizationId: "org_loaded", businessUnitId: "bu_loaded" };
const SWITCHED = { ...LOADED, organizationId: "org_switched", businessUnitId: "bu_switched" };
const SOMEONE_ELSE = { ...LOADED, userId: "usr_someone_else" };

function problemResponse(status: number, body: Record<string, unknown>): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/problem+json" },
  });
}

function staleProblem(params?: Record<string, string>): Record<string, unknown> {
  return {
    type: `${PROBLEM_BASE}session-context-stale`,
    title: "Session Changed",
    status: 409,
    detail: "Your session changed in another tab. Reload this page to continue.",
    ...(params ? { params } : {}),
  };
}

async function load() {
  const sessionContext = await import("@trenova/shared/lib/session-context");
  const { api, ApiRequestError, setCsrfToken } = await import("@trenova/shared/lib/api");
  const { GraphQLRequestError, requestGraphQL } = await import("@trenova/shared/lib/graphql");
  const { isExplainedError } = await import("@trenova/shared/lib/explained-errors");
  setCsrfToken("csrf-token");
  return {
    ...sessionContext,
    api,
    ApiRequestError,
    GraphQLRequestError,
    requestGraphQL,
    isExplainedError,
  };
}

async function loadListening() {
  const loaded = await load();
  const handler = vi.fn();
  loaded.setSessionContextStatusHandler(handler);
  loaded.setSessionContext(LOADED);
  handler.mockClear();
  return { ...loaded, handler };
}

describe("session context", () => {
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.resetModules();
    FakeBroadcastChannel.open = [];
    vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);
    fetchMock = vi.fn(async () => new Response(JSON.stringify({ ok: true }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe("sessionContextOf", () => {
    it("needs the person, the organization and the business unit", async () => {
      const { sessionContextOf } = await load();

      expect(
        sessionContextOf({ id: "usr_1", currentOrganizationId: "org_1", businessUnitId: " bu_1 " }),
      ).toEqual({ userId: "usr_1", organizationId: "org_1", businessUnitId: "bu_1" });
      expect(
        sessionContextOf({ id: "", currentOrganizationId: "org_1", businessUnitId: "bu_1" }),
      ).toBeNull();
      expect(
        sessionContextOf({ id: "usr_1", currentOrganizationId: null, businessUnitId: "bu_1" }),
      ).toBeNull();
      expect(sessionContextOf(null)).toBeNull();
    });
  });

  describe("request headers", () => {
    it("asserts no session until the server has confirmed one", async () => {
      const { api } = await load();

      await api.get("/shipments/");

      const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      const headers = new Headers(init.headers);
      expect(headers.get("X-Trenova-User-Id")).toBeNull();
      expect(headers.get("X-Trenova-Organization-Id")).toBeNull();
      expect(headers.get("X-Trenova-Business-Unit-Id")).toBeNull();
    });

    it("asserts the confirmed session on reads as well as writes", async () => {
      const { api, setSessionContext } = await load();
      setSessionContext(LOADED);

      await api.get("/shipments/");
      await api.post("/shipments/", {});

      for (const [, init] of fetchMock.mock.calls as [string, RequestInit][]) {
        const headers = new Headers(init.headers);
        expect(headers.get("X-Trenova-User-Id")).toBe("usr_loaded");
        expect(headers.get("X-Trenova-Organization-Id")).toBe("org_loaded");
        expect(headers.get("X-Trenova-Business-Unit-Id")).toBe("bu_loaded");
      }
    });

    it("stops asserting a session once it ends", async () => {
      const { api, setSessionContext } = await load();
      setSessionContext(LOADED);
      setSessionContext(null);

      await api.get("/shipments/");

      const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(new Headers(init.headers).get("X-Trenova-User-Id")).toBeNull();
    });
  });

  describe("other tabs", () => {
    it("tells the other tabs once each time the server confirms a new session, and not when it is cleared", async () => {
      const { setSessionContext } = await load();
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      setSessionContext(LOADED);
      setSessionContext({ ...LOADED });
      setSessionContext(SWITCHED);
      setSessionContext(null);

      expect(otherTab.received).toEqual([
        { kind: "confirmed", session: LOADED },
        { kind: "confirmed", session: SWITCHED },
      ]);
    });

    it("tells the other tabs when the session ends", async () => {
      const { announceSignedOut } = await load();
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      announceSignedOut();

      expect(otherTab.received).toEqual([{ kind: "signed-out" }]);
    });

    it("reads each change another tab reports against this tab's session", async () => {
      const { handler } = await loadListening();
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      otherTab.postMessage({ kind: "confirmed", session: SWITCHED });
      otherTab.postMessage({ kind: "confirmed", session: LOADED });
      otherTab.postMessage({ kind: "confirmed", session: SOMEONE_ELSE });
      otherTab.postMessage({ kind: "signed-out" });
      otherTab.postMessage({ kind: "confirmed", session: { userId: "usr_loaded" } });
      otherTab.postMessage({ kind: "unknown" });

      expect(handler.mock.calls).toEqual([
        ["organization-changed"],
        ["current"],
        ["user-changed"],
        ["signed-out"],
      ]);
    });

    it("ignores another tab's news before this tab has a session of its own", async () => {
      const { setSessionContextStatusHandler } = await load();
      const handler = vi.fn();
      setSessionContextStatusHandler(handler);
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      otherTab.postMessage({ kind: "confirmed", session: SWITCHED });
      otherTab.postMessage({ kind: "signed-out" });

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("sign-in elsewhere", () => {
    it("announces a finished sign-in once for each session that reaches the app", async () => {
      const { announceSignedIn, setSessionContext } = await load();
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      announceSignedIn();
      setSessionContext(LOADED);
      announceSignedIn();
      announceSignedIn();
      setSessionContext(SWITCHED);
      announceSignedIn();

      expect(otherTab.received).toEqual([
        { kind: "confirmed", session: LOADED },
        { kind: "signed-in" },
        { kind: "confirmed", session: SWITCHED },
        { kind: "signed-in" },
      ]);
    });

    it("announces again after the session ends and a new one begins", async () => {
      const { announceSignedIn, setSessionContext } = await load();
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      setSessionContext(LOADED);
      announceSignedIn();
      setSessionContext(null);
      setSessionContext(LOADED);
      announceSignedIn();

      expect(otherTab.received).toEqual([
        { kind: "confirmed", session: LOADED },
        { kind: "signed-in" },
        { kind: "confirmed", session: LOADED },
        { kind: "signed-in" },
      ]);
    });

    it("lets a tab without a session follow a sign-in finished in another, until it stops listening", async () => {
      const { onSignedInElsewhere } = await load();
      const follow = vi.fn();
      const stop = onSignedInElsewhere(follow);
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      otherTab.postMessage({ kind: "signed-in" });
      stop();
      otherTab.postMessage({ kind: "signed-in" });

      expect(follow).toHaveBeenCalledTimes(1);
    });

    it("never pulls a tab out of a session of its own", async () => {
      const { onSignedInElsewhere, setSessionContext } = await load();
      setSessionContext(LOADED);
      const follow = vi.fn();
      onSignedInElsewhere(follow);
      const otherTab = new FakeBroadcastChannel(CHANNEL_NAME);

      otherTab.postMessage({ kind: "signed-in" });

      expect(follow).not.toHaveBeenCalled();
    });
  });

  describe("refusals", () => {
    it("reports an organization change and explains the error", async () => {
      const { api, handler, isExplainedError } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem(SWITCHED)));

      const error = await api.post("/shipments/", { proNumber: "1" }).catch((e: unknown) => e);

      expect(handler.mock.calls).toEqual([["organization-changed"]]);
      expect(isExplainedError(error)).toBe(true);
    });

    it("reports someone else signing in", async () => {
      const { api, handler } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem(SOMEONE_ELSE)));

      await api.post("/shipments/", {}).catch(() => undefined);

      expect(handler.mock.calls).toEqual([["user-changed"]]);
    });

    it("keeps the tab current when the refusal names its own session", async () => {
      const { api, handler } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem(LOADED)));

      await api.get("/shipments/").catch(() => undefined);

      expect(handler.mock.calls).toEqual([["current"]]);
    });

    it("leaves a refused save that names its own session to its caller, so the save fails visibly", async () => {
      const { api, handler, isExplainedError } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem(LOADED)));

      const error = await api.post("/shipments/", { proNumber: "1" }).catch((e: unknown) => e);

      expect(handler.mock.calls).toEqual([["current"]]);
      expect(isExplainedError(error)).toBe(false);
      expect(error).toMatchObject({ status: 409 });
    });

    it("treats a refusal that does not name the session as an organization change", async () => {
      const { api, handler } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem()));

      await api.get("/shipments/").catch(() => undefined);

      expect(handler.mock.calls).toEqual([["organization-changed"]]);
    });

    it("recognises the refusal on a GraphQL request", async () => {
      const { handler, requestGraphQL, GraphQLRequestError } = await loadListening();
      fetchMock.mockResolvedValueOnce(problemResponse(409, staleProblem(SWITCHED)));

      const error = await requestGraphQL({
        document: "query Test { ok }",
        operationName: "Test",
      }).catch((e: unknown) => e);

      expect(error).toBeInstanceOf(GraphQLRequestError);
      expect(handler.mock.calls).toEqual([["organization-changed"]]);
    });

    it("leaves every other refusal to its caller", async () => {
      const { api, handler, isExplainedError } = await loadListening();
      fetchMock.mockResolvedValueOnce(
        problemResponse(409, {
          type: `${PROBLEM_BASE}resource-conflict`,
          title: "Resource Conflict",
          status: 409,
        }),
      );

      const error = await api.post("/shipments/", {}).catch((e: unknown) => e);

      expect(handler).not.toHaveBeenCalled();
      expect(isExplainedError(error)).toBe(false);
    });

    it("leaves the refusal to its caller when no app is listening", async () => {
      const { ApiRequestError, reportSessionContextStale, setSessionContext, isExplainedError } =
        await load();
      setSessionContext(LOADED);
      const error = new ApiRequestError(409, {
        type: `${PROBLEM_BASE}session-context-stale`,
        title: "Session Changed",
        status: 409,
        params: SWITCHED,
      });

      expect(reportSessionContextStale(error)).toBe(false);
      expect(isExplainedError(error)).toBe(false);
    });
  });
});
