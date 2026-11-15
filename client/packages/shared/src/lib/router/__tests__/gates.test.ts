import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { isRedirect } from "@tanstack/react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  gates,
  redirectKeepingSearch,
  redirectTo,
  requireGuest,
  type RouteGateContext,
} from "../gates";

function visit(
  pathname: string,
  cause: RouteGateContext["cause"] = "enter",
  searchStr = "",
): RouteGateContext {
  return {
    location: { pathname, searchStr },
    params: {},
    cause,
    abortController: new AbortController(),
  };
}

async function thrownBy(run: () => unknown): Promise<unknown> {
  try {
    await run();
    return undefined;
  } catch (error) {
    return error;
  }
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("gates", () => {
  it("runs every gate in order on the way into a page", async () => {
    const order: string[] = [];
    const gate = gates(
      () => void order.push("session"),
      () => void order.push("permission"),
    );

    await gate(visit("/billing/invoices"));

    expect(order).toEqual(["session", "permission"]);
  });

  it("stops at the first gate that refuses", async () => {
    const later = vi.fn();
    const refusal = new Error("refused");
    const gate = gates(() => {
      throw refusal;
    }, later);

    expect(await thrownBy(() => gate(visit("/billing/invoices")))).toBe(refusal);
    expect(later).not.toHaveBeenCalled();
  });

  it("does not run again while only the query of the same page changes", async () => {
    const check = vi.fn();
    const gate = gates(check);

    await gate(visit("/billing/invoices", "enter", "?status=open"));
    await gate(visit("/billing/invoices", "stay", "?status=paid"));
    await gate(visit("/billing/invoices", "stay", "?status=paid"));

    expect(check).toHaveBeenCalledTimes(1);
  });

  it("runs again when the page under a layout that stays changes", async () => {
    const check = vi.fn();
    const gate = gates(check);

    await gate(visit("/billing/invoices"));
    await gate(visit("/billing/pending-approvals", "stay"));

    expect(check).toHaveBeenCalledTimes(2);
  });

  it("runs again for a page entered afresh", async () => {
    const check = vi.fn();
    const gate = gates(check);

    await gate(visit("/billing/invoices"));
    await gate(visit("/billing/invoices", "enter"));

    expect(check).toHaveBeenCalledTimes(2);
  });

  it("asks again after it refused, rather than remembering the refused page as passed", async () => {
    const check = vi.fn().mockImplementationOnce(() => {
      throw new Error("unreachable");
    });
    const gate = gates(check);

    await thrownBy(() => gate(visit("/billing/invoices")));
    await gate(visit("/billing/invoices", "stay", "?status=open"));

    expect(check).toHaveBeenCalledTimes(2);
  });

  it("does not count a preload as a visit", async () => {
    const check = vi.fn();
    const gate = gates(check);

    await gate(visit("/billing/invoices", "preload"));
    await gate(visit("/billing/invoices", "stay"));

    expect(check).toHaveBeenCalledTimes(2);
  });
});

describe("redirects", () => {
  it("sends a moved address on, replacing it in the history", async () => {
    const thrown = await thrownBy(() => redirectTo("/desk/needs?lens=watchtower")(visit("/x")));

    expect(isRedirect(thrown)).toBe(true);
    expect((thrown as { options: { href?: string; replace?: boolean } }).options).toMatchObject({
      href: "/desk/needs?lens=watchtower",
      replace: true,
    });
  });

  it("carries the query a moved address was opened with", async () => {
    const thrown = await thrownBy(() =>
      redirectKeepingSearch("/hr/workers")(visit("/dispatch/workers", "enter", "?tab=hos")),
    );

    expect((thrown as { options: { href?: string } }).options.href).toBe("/hr/workers?tab=hos");
  });

  it("sends somebody already signed in away from a guest page", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    const thrown = await thrownBy(() => requireGuest("/")(visit("/login")));

    expect((thrown as { options: { href?: string } }).options.href).toBe("/");
  });

  it("lets a guest open a guest page", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(false);

    expect(await thrownBy(() => requireGuest("/")(visit("/login")))).toBeUndefined();
  });
});
