import { IdleSessionGuard } from "@trenova/shared/components/session/idle-session-guard";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { setCsrfToken } from "@trenova/shared/lib/api";
import type { SignOutNotice } from "@trenova/shared/lib/sign-out-notice";
import {
  createSessionActivityClock,
  type SessionActivityClock,
} from "@trenova/shared/lib/session-activity";
import {
  SELF_HOSTED_PUBLIC_CONFIG,
  type PublicConfig,
  type SessionIdlePolicy,
} from "@trenova/shared/types/platform";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The guard reads the app-wide clock, which keeps the latest moment it has seen; each
// test gets a clock of its own so one test's activity cannot move the next one's deadline.
const clockRef = vi.hoisted(() => ({ current: null as SessionActivityClock | null }));

vi.mock("@trenova/shared/lib/session-activity", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/session-activity")>();
  return {
    ...actual,
    sessionActivityClock: {
      last: () => clockRef.current!.last(),
      record: (at: number, options?: { force?: boolean }) => clockRef.current!.record(at, options),
      subscribe: (listener: () => void) => clockRef.current!.subscribe(listener),
    } satisfies SessionActivityClock,
  };
});

const POLICY: SessionIdlePolicy = {
  idleTimeoutSeconds: 30 * 60,
  idleWarningSeconds: 2 * 60,
  keepAliveSeconds: 60,
};
const IDLE_OFF: SessionIdlePolicy = {
  idleTimeoutSeconds: 0,
  idleWarningSeconds: 0,
  keepAliveSeconds: 0,
};
const START = Date.UTC(2026, 9, 16, 14, 0, 0);
const MINUTE = 60_000;

type KeepAliveReply = "ok" | "unauthorized" | "offline";

let keepAliveReply: KeepAliveReply = "ok";
const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
  const url = String(input);
  if (!url.endsWith("/auth/session/keepalive")) {
    throw new Error(`unexpected request to ${url}`);
  }
  switch (keepAliveReply) {
    case "ok":
      return new Response(null, { status: 204 });
    case "unauthorized":
      return new Response(
        JSON.stringify({
          type: "authentication-error",
          title: "Unauthorized",
          status: 401,
          detail: "Session has expired. Please login again.",
        }),
        { status: 401, headers: { "Content-Type": "application/problem+json" } },
      );
    case "offline":
      throw new TypeError("Failed to fetch");
  }
});

function keepAliveCalls(): number {
  return fetchMock.mock.calls.filter(([input]) => String(input).endsWith("/auth/session/keepalive"))
    .length;
}

function renderGuard(policy: SessionIdlePolicy = POLICY) {
  const client = new QueryClient();
  const config: PublicConfig = { ...SELF_HOSTED_PUBLIC_CONFIG, session: policy };
  client.setQueryData(publicConfigQueryOptions.queryKey, config);
  const onSessionEnd = vi.fn<(notice: SignOutNotice | undefined) => void>();

  render(
    <QueryClientProvider client={client}>
      <IdleSessionGuard onSessionEnd={onSessionEnd} />
    </QueryClientProvider>,
  );

  return { onSessionEnd };
}

function advance(ms: number) {
  act(() => {
    vi.advanceTimersByTime(ms);
  });
}

async function flushRequests() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
}

describe("IdleSessionGuard", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(START);
    clockRef.current = createSessionActivityClock({ storage: () => null, target: window });
    keepAliveReply = "ok";
    fetchMock.mockClear();
    vi.stubGlobal("fetch", fetchMock);
    setCsrfToken("csrf-token");
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it("shows nothing until the warning is due", () => {
    renderGuard();

    advance(28 * MINUTE - 1_000);

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("warns two minutes before signing out, counting down", () => {
    renderGuard();

    advance(28 * MINUTE);

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText("Are you still there?")).toBeInTheDocument();
    expect(screen.getByRole("timer")).toHaveTextContent("2:00");

    advance(1_000);
    expect(screen.getByRole("timer")).toHaveTextContent("1:59");
  });

  it("signs out for inactivity when the countdown runs out", () => {
    const { onSessionEnd } = renderGuard();

    advance(30 * MINUTE);

    expect(onSessionEnd).toHaveBeenCalledTimes(1);
    expect(onSessionEnd).toHaveBeenCalledWith("idle");
    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
  });

  it("keeps the session and tells the server when the person stays", async () => {
    const { onSessionEnd } = renderGuard();
    advance(29 * MINUTE);

    fireEvent.click(screen.getByRole("button", { name: "Stay signed in" }));
    await flushRequests();

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(keepAliveCalls()).toBe(1);
    const [, init] = fetchMock.mock.calls[0]!;
    expect(init?.method).toBe("POST");

    advance(28 * MINUTE);
    expect(onSessionEnd).not.toHaveBeenCalled();
  });

  it("ends the session at once, with no notice, when the person signs out from the warning", () => {
    const { onSessionEnd } = renderGuard();
    advance(29 * MINUTE);

    fireEvent.click(screen.getByRole("button", { name: "Sign out" }));

    expect(onSessionEnd).toHaveBeenCalledTimes(1);
    expect(onSessionEnd).toHaveBeenCalledWith(undefined);
  });

  it("sends a keepalive while the person works", async () => {
    renderGuard();

    fireEvent.keyDown(document, { key: "a" });
    await flushRequests();

    expect(keepAliveCalls()).toBe(1);
  });

  it("ends the session when the server says it is already gone", async () => {
    keepAliveReply = "unauthorized";
    const { onSessionEnd } = renderGuard();

    fireEvent.pointerDown(document);
    await flushRequests();

    expect(onSessionEnd).toHaveBeenCalledWith("expired");
  });

  it("keeps the session when a keepalive cannot reach the server", async () => {
    keepAliveReply = "offline";
    const { onSessionEnd } = renderGuard();

    fireEvent.pointerDown(document);
    await flushRequests();

    expect(keepAliveCalls()).toBe(1);
    expect(onSessionEnd).not.toHaveBeenCalled();
  });

  it("never warns, signs out or calls the server while the idle timeout is off", async () => {
    const { onSessionEnd } = renderGuard(IDLE_OFF);

    fireEvent.pointerDown(document);
    advance(24 * 60 * MINUTE);
    await flushRequests();

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    expect(onSessionEnd).not.toHaveBeenCalled();
    expect(keepAliveCalls()).toBe(0);
  });
});
