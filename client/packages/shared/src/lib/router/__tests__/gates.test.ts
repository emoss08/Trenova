import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { isRedirect } from "@tanstack/react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { gates, redirectTo, requireGuest, requireSession, type RouteGateContext } from "../gates";

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

  it("sends somebody already signed in away from a guest page", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    const thrown = await thrownBy(() => requireGuest("/")(visit("/login")));

    expect((thrown as { options: { href?: string } }).options.href).toBe("/");
  });

  it("sends somebody already signed in on to the safe address the sign-in page carries", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    const thrown = await thrownBy(() =>
      requireGuest("/")(
        visit("/login", "enter", "?redirect=%2Fbilling%2Finvoices%3Fstatus%3Dopen"),
      ),
    );

    expect((thrown as { options: { href?: string } }).options.href).toBe(
      "/billing/invoices?status=open",
    );
  });

  it.each([
    ["an absolute address", "?redirect=https%3A%2F%2Fevil.com"],
    ["a protocol-relative address", "?redirect=%2F%2Fevil.com"],
    ["an encoded protocol-relative address", "?redirect=%2F%252F%252Fevil.com"],
    ["the sign-in page itself", "?redirect=%2Flogin"],
  ])("sends somebody already signed in home rather than to %s", async (_, searchStr) => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    const thrown = await thrownBy(() => requireGuest("/")(visit("/login", "enter", searchStr)));

    expect((thrown as { options: { href?: string } }).options.href).toBe("/");
  });

  it("keeps the driver portal's signed-in driver inside the portal", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    const followed = await thrownBy(() =>
      requireGuest("/dash")(visit("/dash/login", "enter", "?redirect=%2Fdash%2Fpay")),
    );
    const refused = await thrownBy(() =>
      requireGuest("/dash")(visit("/dash/login", "enter", "?redirect=%2Fbilling")),
    );

    expect((followed as { options: { href?: string } }).options.href).toBe("/dash/pay");
    expect((refused as { options: { href?: string } }).options.href).toBe("/dash");
  });

  it("lets a guest open a guest page", async () => {
    vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(false);

    expect(await thrownBy(() => requireGuest("/")(visit("/login")))).toBeUndefined();
  });

  it("does not ask the server whether a guest page may open on a preload", async () => {
    const checkAuth = vi.spyOn(useAuthStore.getState(), "checkAuth").mockResolvedValue(true);

    expect(await thrownBy(() => requireGuest("/")(visit("/login", "preload")))).toBeUndefined();
    expect(checkAuth).not.toHaveBeenCalled();
  });
});

describe("requireSession", () => {
  it("sends a signed-out person to sign in, carrying the address they asked for", async () => {
    vi.spyOn(useAuthStore.getState(), "checkSession").mockResolvedValue({ status: "signed_out" });

    const thrown = await thrownBy(() =>
      requireSession({ loginPath: "/login", home: "/" })(
        visit("/billing/invoices", "enter", "?status=open"),
      ),
    );

    expect(isRedirect(thrown)).toBe(true);
    expect((thrown as { options: { href?: string } }).options.href).toBe(
      "/login?redirect=%2Fbilling%2Finvoices%3Fstatus%3Dopen",
    );
  });

  it("keeps the sign-in address clean when the person asked for home", async () => {
    vi.spyOn(useAuthStore.getState(), "checkSession").mockResolvedValue({ status: "signed_out" });

    const thrown = await thrownBy(() =>
      requireSession({ loginPath: "/login", home: "/" })(visit("/")),
    );

    expect((thrown as { options: { href?: string } }).options.href).toBe("/login");
  });

  it("carries the driver portal's page, and not its home, to the portal's sign-in", async () => {
    vi.spyOn(useAuthStore.getState(), "checkSession").mockResolvedValue({ status: "signed_out" });

    const page = await thrownBy(() =>
      requireSession({ loginPath: "/dash/login", home: "/dash" })(visit("/dash/pay/stl_1")),
    );
    const home = await thrownBy(() =>
      requireSession({ loginPath: "/dash/login", home: "/dash" })(visit("/dash")),
    );

    expect((page as { options: { href?: string } }).options.href).toBe(
      "/dash/login?redirect=%2Fdash%2Fpay%2Fstl_1",
    );
    expect((home as { options: { href?: string } }).options.href).toBe("/dash/login");
  });

  it("never asks the server on a preload, and asks on the navigation that follows", async () => {
    const checkSession = vi
      .spyOn(useAuthStore.getState(), "checkSession")
      .mockResolvedValue({ status: "signed_out" });
    const gate = gates(requireSession({ loginPath: "/login", home: "/" }));

    expect(await thrownBy(() => gate(visit("/billing/invoices", "preload")))).toBeUndefined();
    expect(checkSession).not.toHaveBeenCalled();

    const thrown = await thrownBy(() => gate(visit("/billing/invoices", "enter")));

    expect(checkSession).toHaveBeenCalledTimes(1);
    expect((thrown as { options: { href?: string } }).options.href).toBe(
      "/login?redirect=%2Fbilling%2Finvoices",
    );
  });
});
