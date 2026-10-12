import { describe, expect, it } from "vitest";
import { customerStatementPath } from "@/lib/ar-links";

describe("customerStatementPath", () => {
  it("opens the customer's statement, with an id that is not URL-safe encoded", () => {
    expect(customerStatementPath("cus_1")).toBe("/accounting/ar/customer-statement/cus_1");
    expect(customerStatementPath("a/b")).toBe("/accounting/ar/customer-statement/a%2Fb");
  });
});
