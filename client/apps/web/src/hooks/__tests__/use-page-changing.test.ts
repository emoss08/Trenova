import { describe, expect, it } from "vitest";
import { isChangingPage } from "../use-page-changing";

describe("isChangingPage", () => {
  it("is true while a navigation to another path is pending", () => {
    expect(
      isChangingPage({
        status: "pending",
        location: { pathname: "/customers" },
        resolvedLocation: { pathname: "/shipments" },
      }),
    ).toBe(true);
  });

  it("is false while only the query of the page on screen changes", () => {
    expect(
      isChangingPage({
        status: "pending",
        location: { pathname: "/shipments" },
        resolvedLocation: { pathname: "/shipments" },
      }),
    ).toBe(false);
  });

  it("is false once the router is idle", () => {
    expect(
      isChangingPage({
        status: "idle",
        location: { pathname: "/customers" },
        resolvedLocation: { pathname: "/customers" },
      }),
    ).toBe(false);
  });

  it("is false for the first page, which its own loading screen covers", () => {
    expect(isChangingPage({ status: "pending", location: { pathname: "/" } })).toBe(false);
  });
});
