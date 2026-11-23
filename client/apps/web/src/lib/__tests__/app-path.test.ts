import { appPath, isAppPath } from "@/lib/app-path";
import type { RecordLinkPath } from "@/config/record-links";
import { describe, expect, it } from "vitest";

describe("appPath", () => {
  it("returns a registered page's path as it is written", () => {
    expect(appPath("/hr/workers")).toBe("/hr/workers");
    expect(appPath("/")).toBe("/");
  });

  it("puts each param into its own segment", () => {
    expect(appPath("/billing/configuration-files/formula-templates/$id/edit", { id: "ft_1" })).toBe(
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
    expect(appPath("/desk/r/$kind/$id", { kind: "shipment", id: "shp_1" })).toBe(
      "/desk/r/shipment/shp_1",
    );
  });

  it("encodes a param so it stays one segment", () => {
    expect(appPath("/desk/r/$kind/$id", { kind: "a b", id: "x/y?z=1&w#v" })).toBe(
      "/desk/r/a%20b/x%2Fy%3Fz%3D1%26w%23v",
    );
  });

  it("does not read a param's value as another param", () => {
    expect(appPath("/desk/r/$kind/$id", { kind: "$id", id: "1" })).toBe("/desk/r/%24id/1");
  });

  it("refuses a path the router does not register", () => {
    // @ts-expect-error: no such page.
    appPath("/shipments");
    // @ts-expect-error: a pattern needs its params.
    appPath("/desk/r/$kind/$id");
    // @ts-expect-error: every param of the pattern is required.
    appPath("/desk/r/$kind/$id", { kind: "shipment" });
    // @ts-expect-error: a record link names a registered page.
    const unregistered: RecordLinkPath = "/shipments";
    const registered: RecordLinkPath = "/reports/explore/{id}";

    expect([unregistered, registered]).toHaveLength(2);
  });
});

describe("isAppPath", () => {
  it.each([
    "/",
    "/billing/invoices",
    "/hr/workers/wrk_1?tab=qualification&view=credentials",
    "/billing/invoices?next=//example.com",
    "/search#//example.com",
    "/search?q=100%",
  ])("accepts %j", (href) => {
    expect(isAppPath(href)).toBe(true);
  });

  it.each([
    "",
    "billing/invoices",
    "//evil.example/x",
    "///evil.example",
    "/\\evil.example",
    "/\t/evil.example",
    "/\n/evil.example",
    "/\r\n/evil.example",
    "/\t\\evil.example",
    "https://example.com",
    "javascript:alert(1)",
    null,
    undefined,
  ])("refuses %j", (href) => {
    expect(isAppPath(href)).toBe(false);
  });
});
