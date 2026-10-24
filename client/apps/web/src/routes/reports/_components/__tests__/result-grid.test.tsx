import { render, screen, within } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReportPreviewColumn } from "@/lib/graphql/reports";
import { REPORT_PAGE_ROWS } from "../report-cells";
import { ResultGrid } from "../result-grid";

// happy-dom lays nothing out, so the virtualiser would draw no rows. This stand-in draws the
// first screen of rows, as the real one does in a viewport of that height.
const SCREEN_ROWS = 30;
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({ count, estimateSize }: { count: number; estimateSize: () => number }) => ({
    getTotalSize: () => count * estimateSize(),
    getVirtualItems: () =>
      Array.from({ length: Math.min(count, SCREEN_ROWS) }, (_, index) => ({
        index,
        key: index,
        start: index * estimateSize(),
        size: estimateSize(),
      })),
  }),
}));

const DISPLAY = {
  style: "number",
  decimals: 2,
  grouping: true,
  currency: "USD",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "dateTime",
  boolStyle: "yesNo",
  durationUnit: "seconds",
  durationStyle: "clock",
  nullText: "",
  rules: [{ op: "gt", value: 1000, upper: 0, tone: "warning" }],
  band: null,
};

const COLUMNS = [
  {
    id: "lane",
    label: "Lane",
    type: "string",
    format: null,
    display: { ...DISPLAY, style: "text", rules: [] },
  },
  { id: "revenue", label: "Revenue", type: "decimal", format: null, display: DISPLAY },
] as unknown as ReportPreviewColumn[];

function formatCalls(spy: { mock: { calls: unknown[][] } }): number {
  return spy.mock.calls.filter(([op]) => op === "report.format").length;
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("ResultGrid", () => {
  // GraphQL sends decimals as strings; the module rounds the digits as written, so 1.005
  // shows as 1.01 where the old float formatting wrote 1.00 (testdata/report.json).
  it("shows each cell as the rules module formats it, with its tone", async () => {
    render(
      <ResultGrid
        columns={COLUMNS}
        rows={[
          ["TX-GA", "1.005"],
          ["IL-OH", "2500.5"],
        ]}
      />,
    );

    expect(await screen.findByTitle("1.01")).toHaveTextContent("1.01");
    const toned = screen.getByTitle("2,500.50");
    expect(toned).toHaveClass("text-warning-foreground");
    expect(screen.getByTitle("TX-GA")).toBeInTheDocument();
  });

  it("formats a page of rows per call, and every page once when filtering", async () => {
    const spy = vi.spyOn(rulesSync(), "call");
    const rows = Array.from({ length: REPORT_PAGE_ROWS * 2 + 5 }, (_, index) => [
      `Lane ${index}`,
      String(index),
    ]);
    const totals = [null, "15"];

    const { rerender } = render(<ResultGrid columns={COLUMNS} rows={rows} totals={totals} />);
    await screen.findByTitle("Lane 0");
    // One call for the first page, one for the totals row.
    expect(formatCalls(spy)).toBe(2);

    rerender(<ResultGrid columns={COLUMNS} rows={rows} totals={totals} quickFilter="lane 404" />);
    // Filtering reads every row: the two pages not yet formatted, one call each.
    expect(formatCalls(spy)).toBe(4);
    const grid = await screen.findByTitle("Lane 404");
    expect(within(grid.parentElement as HTMLElement).getByTitle("404.00")).toBeInTheDocument();
    expect(screen.queryByTitle("Lane 0")).not.toBeInTheDocument();
  });

  it("names the totals row and formats its values", async () => {
    render(<ResultGrid columns={COLUMNS} rows={[["TX-GA", "1"]]} totals={[null, "1234.5"]} />);
    expect(await screen.findByTitle("Total")).toBeInTheDocument();
    expect(screen.getByTitle("1,234.50")).toBeInTheDocument();
  });

  // A date bucket is the instant its period starts in the organization's zone. Kiritimati is
  // UTC+14, so its March starts on Feb 28 in every zone a browser can be in: only writing the
  // date in the result's zone shows the period the server grouped.
  it("writes a date bucket in the result's zone, not the browser's", async () => {
    const monthBucket = [
      {
        id: "period",
        label: "Month",
        type: "epoch",
        format: null,
        display: { ...DISPLAY, style: "date", dateStyle: "monthYear", rules: [] },
      },
    ] as unknown as ReportPreviewColumn[];
    const marchInKiritimati = Date.UTC(2026, 1, 28, 10, 0, 0) / 1000;

    render(
      <ResultGrid
        columns={monthBucket}
        rows={[[marchInKiritimati]]}
        timeZone="Pacific/Kiritimati"
      />,
    );

    expect(await screen.findByTitle("Mar 2026")).toHaveTextContent("Mar 2026");
  });
});
