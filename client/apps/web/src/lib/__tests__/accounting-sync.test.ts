import { describe, expect, it } from "vitest";
import {
  ACCOUNTING_RECONNECT_WARNING_SECONDS,
  accountingConnectionPhase,
  accountingSetupPath,
  hasLiveAccountingConnection,
  isTrustedAuthorizeUrl,
  readAccountingCallback,
  reconnectDeadlineNear,
} from "../accounting-sync";

const DAY = 24 * 60 * 60;

describe("accountingConnectionPhase", () => {
  it("reads a connection that answers as complete", () => {
    expect(accountingConnectionPhase("Connected")).toBe("complete");
  });

  it("reads a connection that is retrying as needing attention, not as failed", () => {
    expect(accountingConnectionPhase("Degraded")).toBe("attention");
  });

  it("reads a failing or revoked connection as failed", () => {
    expect(accountingConnectionPhase("Failing")).toBe("failed");
    expect(accountingConnectionPhase("Revoked")).toBe("failed");
  });

  it("reads a deliberate disconnect as closed rather than failed", () => {
    expect(accountingConnectionPhase("Disconnected")).toBe("closed");
  });
});

describe("hasLiveAccountingConnection", () => {
  it("has none before the first connect", () => {
    expect(hasLiveAccountingConnection(null)).toBe(false);
  });

  it("has none once someone disconnected it", () => {
    expect(hasLiveAccountingConnection({ status: "Disconnected" })).toBe(false);
  });

  it("keeps a revoked connection, which needs a reconnect rather than a fresh setup", () => {
    expect(hasLiveAccountingConnection({ status: "Revoked" })).toBe(true);
  });

  it("keeps a failing connection", () => {
    expect(hasLiveAccountingConnection({ status: "Failing" })).toBe(true);
  });
});

describe("isTrustedAuthorizeUrl", () => {
  it("accepts Intuit's authorize page for QuickBooks", () => {
    expect(
      isTrustedAuthorizeUrl(
        "QuickBooksOnline",
        "https://appcenter.intuit.com/connect/oauth2?client_id=abc&state=xyz",
      ),
    ).toBe(true);
  });

  it("refuses another host, even one that ends with the right name", () => {
    expect(
      isTrustedAuthorizeUrl("QuickBooksOnline", "https://appcenter.intuit.com.example.org/connect"),
    ).toBe(false);
    expect(isTrustedAuthorizeUrl("QuickBooksOnline", "https://evilappcenter.intuit.com/")).toBe(
      false,
    );
  });

  it("refuses plain http and script URLs", () => {
    expect(isTrustedAuthorizeUrl("QuickBooksOnline", "http://appcenter.intuit.com/connect")).toBe(
      false,
    );
    expect(isTrustedAuthorizeUrl("QuickBooksOnline", "javascript:alert(1)")).toBe(false);
  });

  it("refuses a URL that carries credentials in front of the host", () => {
    expect(
      isTrustedAuthorizeUrl("QuickBooksOnline", "https://appcenter.intuit.com@example.org/"),
    ).toBe(false);
    expect(
      isTrustedAuthorizeUrl("QuickBooksOnline", "https://user:pass@appcenter.intuit.com/"),
    ).toBe(false);
  });

  it("refuses text that is not a URL", () => {
    expect(isTrustedAuthorizeUrl("QuickBooksOnline", "")).toBe(false);
    expect(isTrustedAuthorizeUrl("QuickBooksOnline", "not a url")).toBe(false);
  });
});

describe("readAccountingCallback", () => {
  it("reads what the provider sends back after approval", () => {
    expect(readAccountingCallback(new URLSearchParams("code=c1&state=s1&realmId=9130348"))).toEqual(
      { kind: "authorized", code: "c1", state: "s1", realmId: "9130348" },
    );
  });

  it("trims stray whitespace from every value", () => {
    expect(
      readAccountingCallback(new URLSearchParams("code=%20c1%20&state=s1%0A&realmId=%209")),
    ).toEqual({ kind: "authorized", code: "c1", state: "s1", realmId: "9" });
  });

  it("reports a refusal when the person declined, whatever else came with it", () => {
    expect(
      readAccountingCallback(new URLSearchParams("error=access_denied&state=s1&code=c1")),
    ).toEqual({ kind: "denied" });
  });

  it("reports any other provider error as a provider error", () => {
    expect(readAccountingCallback(new URLSearchParams("error=invalid_scope&state=s1"))).toEqual({
      kind: "provider-error",
      error: "invalid_scope",
    });
  });

  it("reports a return missing any of the three values as incomplete", () => {
    expect(readAccountingCallback(new URLSearchParams("code=c1&state=s1"))).toEqual({
      kind: "incomplete",
    });
    expect(readAccountingCallback(new URLSearchParams("state=s1&realmId=9"))).toEqual({
      kind: "incomplete",
    });
    expect(readAccountingCallback(new URLSearchParams("code=c1&realmId=9&state="))).toEqual({
      kind: "incomplete",
    });
    expect(readAccountingCallback(new URLSearchParams(""))).toEqual({ kind: "incomplete" });
  });
});

describe("reconnectDeadlineNear", () => {
  const now = 1_800_000_000;

  it("is not near while the limit is further off than the warning window", () => {
    expect(reconnectDeadlineNear(now + ACCOUNTING_RECONNECT_WARNING_SECONDS + DAY, now)).toBe(
      false,
    );
  });

  it("is near inside the warning window", () => {
    expect(reconnectDeadlineNear(now + 3 * DAY, now)).toBe(true);
  });

  it("is near once the limit has passed", () => {
    expect(reconnectDeadlineNear(now - DAY, now)).toBe(true);
  });

  it("warns fourteen days out, the same window the server raises its Watchtower item at", () => {
    expect(ACCOUNTING_RECONNECT_WARNING_SECONDS).toBe(14 * DAY);
  });
});

describe("accountingSetupPath", () => {
  it("opens the integrations page with the system's dialog open", () => {
    expect(accountingSetupPath("QuickBooksOnline")).toBe(
      "/admin/integrations?type=QuickBooksOnline",
    );
  });
});
