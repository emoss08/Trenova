import { describe, expect, it } from "vitest";
import { groupByReportCategory, reportCategoryLabel } from "./reports-page-state";

describe("groupByReportCategory", () => {
  it("heads every category a stock report uses with its capitalized name", () => {
    const groups = groupByReportCategory(
      [
        "Accounting",
        "Billing",
        "Compliance",
        "Customers",
        "Desk",
        "Fleet",
        "Operations",
        "Payroll",
      ].map((category) => ({ category })),
    );

    expect(groups.map((group) => group.label)).toEqual([
      "Operations",
      "Billing",
      "Accounting",
      "Payroll",
      "Fleet",
      "Compliance",
      "Customers",
      "Desk",
    ]);
  });

  it("puts a stock report and a saved report of one category under one heading", () => {
    const groups = groupByReportCategory([{ category: "Payroll" }, { category: "payroll" }]);

    expect(groups).toEqual([
      {
        key: "payroll",
        label: "Payroll",
        items: [{ category: "Payroll" }, { category: "payroll" }],
      },
    ]);
  });

  it("heads a category the client has no name for with the text the report carries", () => {
    const groups = groupByReportCategory([{ category: "Safety" }]);

    expect(groups).toEqual([{ key: "safety", label: "Safety", items: [{ category: "Safety" }] }]);
  });
});

describe("reportCategoryLabel", () => {
  it("names a report without a category Uncategorized", () => {
    expect(reportCategoryLabel("")).toBe("Uncategorized");
  });
});
