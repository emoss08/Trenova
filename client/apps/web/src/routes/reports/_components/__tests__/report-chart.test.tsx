import { render, screen } from "@testing-library/react";
import type { ReportChartSpec } from "@/types/report";
import { describe, expect, it } from "vitest";
import { ReportChart, type ChartColumn } from "../report-chart";

const MONEY = {
  style: "currency",
  decimals: 2,
  grouping: true,
  currency: "USD",
  negative: "parentheses",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "dateTime",
  boolStyle: "yesNo",
  durationUnit: "seconds",
  durationStyle: "clock",
  nullText: "",
  rules: [],
  band: null,
};

describe("ReportChart", () => {
  // A KPI shows its measure through the column's display, rounded by the rules module on the
  // digits GraphQL sent: 2.675 is $2.68, where float formatting wrote $2.67.
  it("writes a KPI's value the way the grid and the exports do", async () => {
    const columns: ChartColumn[] = [
      { id: "revenue", label: "Revenue", type: "decimal", display: MONEY },
      { id: "previous", label: "Last month", type: "decimal", display: MONEY },
    ];
    render(
      <ReportChart
        chart={{ id: "kpi", type: "kpi", seriesIds: ["revenue"] } as unknown as ReportChartSpec}
        columns={columns}
        rows={[["2.675", "-1.005"]]}
      />,
    );

    expect(await screen.findByText("$2.68")).toBeInTheDocument();
    expect(screen.getByText("Revenue")).toBeInTheDocument();
  });
});
