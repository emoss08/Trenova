import {
  signInPathAfter,
  signOutNoticeFrom,
  withoutSignOutNotice,
} from "@trenova/shared/lib/sign-out-notice";
import { describe, expect, it } from "vitest";

describe("sign-out notices", () => {
  it("lands on the plain sign-in page when there is nothing to say", () => {
    expect(signInPathAfter("/login")).toBe("/login");
  });

  it("carries the reason to the sign-in page", () => {
    expect(signInPathAfter("/login", "idle")).toBe("/login?signed_out=idle");
    expect(signInPathAfter("/dash/login", "expired")).toBe("/dash/login?signed_out=expired");
  });

  it("carries the address to return to, beside the reason or alone", () => {
    expect(signInPathAfter("/login", "expired", "/billing/invoices?status=open")).toBe(
      "/login?signed_out=expired&redirect=%2Fbilling%2Finvoices%3Fstatus%3Dopen",
    );
    expect(signInPathAfter("/login", undefined, "/billing/invoices")).toBe(
      "/login?redirect=%2Fbilling%2Finvoices",
    );
    expect(signInPathAfter("/login", "expired", null)).toBe("/login?signed_out=expired");
  });

  it("keeps the return address when the notice is dropped", () => {
    const next = withoutSignOutNotice(
      new URLSearchParams("signed_out=expired&redirect=%2Fbilling"),
    );

    expect(next.toString()).toBe("redirect=%2Fbilling");
  });

  it("reads back the notices it writes", () => {
    expect(signOutNoticeFrom(new URLSearchParams("signed_out=idle"))).toBe("idle");
    expect(signOutNoticeFrom(new URLSearchParams("signed_out=expired"))).toBe("expired");
    expect(signOutNoticeFrom(new URLSearchParams("signed_out=elsewhere"))).toBe("elsewhere");
  });

  it("ignores a missing or unknown notice", () => {
    expect(signOutNoticeFrom(new URLSearchParams(""))).toBeNull();
    expect(signOutNoticeFrom(new URLSearchParams("signed_out=hacked"))).toBeNull();
    expect(signOutNoticeFrom(new URLSearchParams("signed_out="))).toBeNull();
  });

  it("drops only the notice from the search", () => {
    const next = withoutSignOutNotice(new URLSearchParams("signed_out=idle&sso_error=x"));

    expect(next.toString()).toBe("sso_error=x");
  });
});
