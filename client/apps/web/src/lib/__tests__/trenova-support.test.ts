import { describe, expect, it } from "vitest";
import { newSupportRequestUrl, supportRequestPage } from "../trenova-support";

describe("supportRequestPage", () => {
  it("is the route's pattern, never a record's id", () => {
    expect(supportRequestPage("/shipments/$shipmentId")).toBe("/shipments/$shipmentId");
    expect(supportRequestPage("/dispatch-console/")).toBe("/dispatch-console");
    expect(supportRequestPage("/")).toBe("/");
  });

  it("leaves out what the console would refuse", () => {
    expect(supportRequestPage(undefined)).toBeUndefined();
    expect(supportRequestPage("")).toBeUndefined();
    expect(supportRequestPage("shipments")).toBeUndefined();
    expect(supportRequestPage("/two words")).toBeUndefined();
    expect(supportRequestPage("/a?b=1")).toBeUndefined();
    expect(supportRequestPage(`/${"a".repeat(254)}`)).toBeUndefined();
  });
});

describe("newSupportRequestUrl", () => {
  const form = "https://console.example.com/support/new?from=trenova&instance=i-1";

  it("adds the page to the form's own parameters", () => {
    expect(newSupportRequestUrl(form, "/shipments/$shipmentId")).toBe(
      "https://console.example.com/support/new?from=trenova&instance=i-1&page=%2Fshipments%2F%24shipmentId",
    );
  });

  it("keeps the form as it is without a page", () => {
    expect(newSupportRequestUrl(form, undefined)).toBe(form);
  });
});
