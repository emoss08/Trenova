import { beforeEach, describe, expect, it, vi } from "vitest";
import { requireWebSession } from "../session-gate";

const mocks = vi.hoisted(() => ({ requireSession: vi.fn(), clearSessionCache: vi.fn() }));

vi.mock("@trenova/shared/lib/session-loader", () => ({ requireSession: mocks.requireSession }));
vi.mock("@/lib/session-cache", () => ({ clearSessionCache: mocks.clearSessionCache }));

beforeEach(() => {
  mocks.requireSession.mockReset();
  mocks.clearSessionCache.mockReset();
  mocks.clearSessionCache.mockResolvedValue(undefined);
});

describe("requireWebSession", () => {
  it("clears what the browser kept for a session the server refused, then signs in", async () => {
    const toLogin = new Response(null, { status: 302, headers: { Location: "/login" } });
    const calls: string[] = [];
    mocks.requireSession.mockImplementation(async () => {
      calls.push("check session");
      return toLogin;
    });
    mocks.clearSessionCache.mockImplementation(async () => void calls.push("clear cache"));

    await expect(requireWebSession("/login")).resolves.toBe(toLogin);

    expect(mocks.requireSession).toHaveBeenCalledWith("/login");
    expect(calls).toEqual(["check session", "clear cache"]);
  });

  it("keeps a signed-in session's state", async () => {
    mocks.requireSession.mockResolvedValue(null);

    await expect(requireWebSession("/login")).resolves.toBeNull();

    expect(mocks.clearSessionCache).not.toHaveBeenCalled();
  });

  it("keeps state when the check could not reach a verdict", async () => {
    const unreachable = new Error("Too many requests");
    mocks.requireSession.mockRejectedValue(unreachable);

    await expect(requireWebSession("/login")).rejects.toBe(unreachable);

    expect(mocks.clearSessionCache).not.toHaveBeenCalled();
  });
});
