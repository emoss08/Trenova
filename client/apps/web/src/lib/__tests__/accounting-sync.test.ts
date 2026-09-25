import { describe, expect, it } from "vitest";
import {
  ACCOUNTING_MAPPINGS_PATH,
  ACCOUNTING_RECONNECT_WARNING_SECONDS,
  accountingConnectionPhase,
  accountingReferenceDetail,
  accountingMappingCreatable,
  accountingMappingFilterKey,
  accountingMappingPhase,
  accountingMappingRecordPath,
  accountingSetupPath,
  needsAccountingMappings,
  precheckedMappingIds,
  referenceRefreshRunning,
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

describe("accountingMappingPhase", () => {
  it("reads an unmatched mapping as needing attention", () => {
    expect(accountingMappingPhase("Unmatched")).toBe("attention");
  });

  it("reads a proposal as waiting on a person", () => {
    expect(accountingMappingPhase("Proposed")).toBe("awaiting");
  });

  it("reads a confirmed mapping as complete", () => {
    expect(accountingMappingPhase("Confirmed")).toBe("complete");
  });
});

describe("referenceRefreshRunning", () => {
  it("is running while a start is recorded", () => {
    expect(referenceRefreshRunning({ referenceRefreshStartedAt: 100 })).toBe(true);
  });

  it("is not running once the server cleared the start", () => {
    expect(referenceRefreshRunning({ referenceRefreshStartedAt: null })).toBe(false);
    expect(referenceRefreshRunning(null)).toBe(false);
  });
});

describe("needsAccountingMappings", () => {
  it("keeps a live connection in setup until the mappings step is finished", () => {
    expect(needsAccountingMappings({ status: "Connected", setupStep: "Mappings" })).toBe(true);
    expect(needsAccountingMappings({ status: "Degraded", setupStep: "Mappings" })).toBe(true);
  });

  it("leaves a finished or disconnected connection alone", () => {
    expect(needsAccountingMappings({ status: "Connected", setupStep: "Complete" })).toBe(false);
    expect(needsAccountingMappings({ status: "Disconnected", setupStep: "Mappings" })).toBe(false);
    expect(needsAccountingMappings(null)).toBe(false);
  });
});

describe("precheckedMappingIds", () => {
  it("ticks only the proposals the server marked sure enough", () => {
    expect(
      precheckedMappingIds([
        { id: "a", state: "Proposed", prechecked: true },
        { id: "b", state: "Proposed", prechecked: false },
        { id: "c", state: "Confirmed", prechecked: false },
        { id: "d", state: "Unmatched", prechecked: false },
      ]),
    ).toEqual(["a"]);
  });
});

describe("accountingMappingRecordPath", () => {
  it("opens the Trenova customer or carrier behind a mapping", () => {
    expect(accountingMappingRecordPath({ targetType: "Customer", trenovaObjectId: "cus_1" })).toBe(
      "/billing/configuration-files/customers?panelType=edit&panelEntityId=cus_1",
    );
    expect(accountingMappingRecordPath({ targetType: "Carrier", trenovaObjectId: "car_1" })).toBe(
      "/dispatch/carriers?panelType=edit&panelEntityId=car_1",
    );
  });

  it("has nothing to open for a setting", () => {
    expect(accountingMappingRecordPath({ targetType: "AccountRole", trenovaObjectId: null })).toBe(
      null,
    );
    expect(
      accountingMappingRecordPath({ targetType: "AccessorialCharge", trenovaObjectId: "acc_1" }),
    ).toBe(null);
  });
});

describe("accountingMappingFilterKey", () => {
  it("names the same filter the same way whatever order it was built in", () => {
    expect(
      accountingMappingFilterKey({ states: ["Proposed", "Unmatched"], targetTypes: ["Customer"] }),
    ).toBe(
      accountingMappingFilterKey({ targetTypes: ["Customer"], states: ["Unmatched", "Proposed"] }),
    );
  });

  it("tells different filters apart", () => {
    expect(accountingMappingFilterKey({ search: "acme" })).not.toBe(
      accountingMappingFilterKey({ search: "acm" }),
    );
  });
});

describe("ACCOUNTING_MAPPINGS_PATH", () => {
  it("lives under the accounting module", () => {
    expect(ACCOUNTING_MAPPINGS_PATH).toBe("/accounting/sync/mappings");
  });
});

describe("accountingMappingCreatable", () => {
  it("offers to create items, customers and vendors that are not confirmed yet", () => {
    expect(accountingMappingCreatable({ providerKind: "Item", state: "Unmatched" })).toBe(true);
    expect(accountingMappingCreatable({ providerKind: "Vendor", state: "Proposed" })).toBe(true);
    expect(accountingMappingCreatable({ providerKind: "Customer", state: "Unmatched" })).toBe(true);
  });

  it("never offers to create the bookkeeper's records or a second record", () => {
    expect(accountingMappingCreatable({ providerKind: "Account", state: "Unmatched" })).toBe(false);
    expect(accountingMappingCreatable({ providerKind: "Term", state: "Unmatched" })).toBe(false);
    expect(accountingMappingCreatable({ providerKind: "PaymentMethod", state: "Unmatched" })).toBe(
      false,
    );
    expect(accountingMappingCreatable({ providerKind: "Item", state: "Confirmed" })).toBe(false);
  });
});

describe("accountingReferenceDetail", () => {
  it("describes an account by its type and number", () => {
    expect(
      accountingReferenceDetail({
        accountType: "Income",
        itemType: "",
        number: "4000",
        companyName: "",
        city: "",
        state: "",
      }),
    ).toBe("Income · 4000");
  });

  it("describes a vendor by where it is", () => {
    expect(
      accountingReferenceDetail({
        accountType: "",
        itemType: "",
        number: "",
        companyName: "Roadrunner Freight LLC",
        city: "Dallas",
        state: "TX",
      }),
    ).toBe("Roadrunner Freight LLC · Dallas · TX");
  });
});
