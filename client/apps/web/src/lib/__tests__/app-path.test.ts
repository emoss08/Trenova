import { describe, expect, it } from "vitest";
import { isAppPath } from "../app-path";

describe("isAppPath", () => {
  it.each(["/", "/billing/invoices", "/hr/workers?panelType=edit&panelEntityId=wrk_1"])(
    "accepts %j",
    (href) => {
      expect(isAppPath(href)).toBe(true);
    },
  );

  it.each([
    "",
    "billing/invoices",
    "//evil.example/x",
    "/\\evil.example",
    "https://example.com",
    "javascript:alert(1)",
    null,
    undefined,
  ])("refuses %j", (href) => {
    expect(isAppPath(href)).toBe(false);
  });
});
