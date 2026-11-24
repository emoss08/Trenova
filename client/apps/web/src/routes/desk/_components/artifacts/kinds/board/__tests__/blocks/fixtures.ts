import type { DeskBoardMetric, DeskBoardRow, DeskBoardValue } from "@/lib/graphql/desk-board-card";
import type {
  DeskBoardChart,
  DeskBoardChartRow,
  DeskBoardChartTick,
} from "../../blocks/metric-types";

/*
Fixtures written from the GraphQL contract (desk_board.graphqls: DeskBoardMetric and its
parts), not from what the renderers produce: every list present, every nullable field null
unless a test sets it, decimals as strings, moments as unix seconds.
*/

export const NOW = 1_795_000_000;
export const DAY = 86_400;

export function value(kind: string, number: string | number, currency = ""): DeskBoardValue {
  return { kind, number: String(number), currency, text: "" };
}
export const usd = (amount: string | number) => value("money", amount, "USD");
export const pct = (fraction: number) => value("percent", fraction);
export const count = (n: number) => value("count", n);

export function metric(fields: Partial<DeskBoardMetric> = {}): DeskBoardMetric {
  return {
    name: "",
    form: "figures",
    title: "",
    source: "",
    figures: [],
    chart: null,
    split: null,
    grid: null,
    log: [],
    checks: [],
    table: null,
    ...fields,
  };
}

export function chart(fields: Partial<DeskBoardChart> = {}): DeskBoardChart {
  return {
    unit: "number",
    currency: "",
    ticks: [],
    series: [],
    marked: [],
    band: null,
    target: null,
    ref: null,
    ghost: false,
    rows: [],
    ...fields,
  };
}

export const tick = (label: string, marked = false): DeskBoardChartTick => ({
  at: null,
  label,
  marked,
});

export function series(name: string, role: string, points: (number | null)[]) {
  return { name, role, points: points.map((point) => (point === null ? null : String(point))) };
}

export function chartRow(fields: Partial<DeskBoardChartRow> = {}): DeskBoardChartRow {
  return {
    label: "",
    record: null,
    value: null,
    marked: false,
    rest: false,
    spans: [],
    dots: [],
    ...fields,
  };
}

export function row(kind: string, id: string, label: string): DeskBoardRow {
  return {
    kind,
    id,
    readable: true,
    label,
    subtitle: "",
    status: "",
    statusTone: "",
    opens: null,
    measure: null,
  };
}

/** The OnTime frame's trend: nine weeks, the slip marked with a band, this week a ghost. */
export function onTimeChart(fields: Partial<DeskBoardChart> = {}): DeskBoardChart {
  return chart({
    unit: "percent",
    ticks: [
      "Sep 28",
      "Oct 5",
      "Oct 12",
      "Oct 19",
      "Oct 26",
      "Nov 2",
      "Nov 9",
      "Nov 16",
      "Nov 23",
    ].map((label) => tick(label)),
    series: [
      series("On time", "now", [0.938, 0.941, 0.935, 0.944, 0.91, 0.882, 0.906, 0.924, 0.95]),
    ],
    marked: [5],
    band: { from: 5, to: 5, low: null, high: null },
    target: { value: "0.92", label: "Target" },
    ghost: true,
    ...fields,
  });
}

/** The Aging frame: last month light, this month dark, the 31 to 60 bucket marked. */
export function agingChart(): DeskBoardChart {
  return chart({
    unit: "money",
    currency: "USD",
    ticks: ["Current", "1–30", "31–60", "61–90", "90+"].map((label, index) =>
      tick(label, index === 2),
    ),
    series: [
      series("Oct", "was", [398700, 151200, 41650, 18900, 11200]),
      series("Nov", "now", [412300, 168450, 74920, 21300, 9840]),
    ],
    marked: [2],
  });
}

/** The OnTime frame's ranked bars: late deliveries by customer, the largest marked. */
export function lateChart(): DeskBoardChart {
  return chart({
    unit: "count",
    rows: [
      chartRow({ label: "Northline Grocers", value: count(17), marked: true }),
      chartRow({ label: "Acme Foods", value: count(7) }),
      chartRow({ label: "Harbor Paper Co.", value: count(4) }),
      chartRow({ label: "Delta Tile Supply", value: count(2) }),
      chartRow({ label: "3 other customers", value: count(2) }),
    ],
  });
}

/** The Settlements frame's five weekly batches, net below and deductions above. */
export function weeksChart(marked: number[] = [4]): DeskBoardChart {
  return chart({
    unit: "money",
    currency: "USD",
    ticks: ["Oct 19", "Oct 26", "Nov 2", "Nov 9", "Nov 16"].map((label) => tick(label)),
    series: [
      series("net", "stack", [111304.81, 118220.23, 108473.23, 119132.2, 121415.9]),
      series("deductions", "stack", [19935.19, 20684.77, 19186.77, 20987.8, 21444.1]),
    ],
    marked,
  });
}

/** The Facilities frame's columns: the median stop by arrival window against 2h free. */
export function arrivalChart(points: (number | null)[] = [7800, 10200, 15600, 14700, 11400]) {
  return chart({
    unit: "duration",
    ticks: ["4 AM", "6 AM", "8 AM", "10 AM", "12 PM"].map((label, index) =>
      tick(label, index === 2),
    ),
    series: [series("Median stop", "now", points)],
    marked: [2],
    ref: { value: "7200", label: "Free" },
  });
}

/** The FuelCost frame's tractors: quiet dots inside the band, outliers marked and named. */
export function tractorsChart(): DeskBoardChart {
  return chart({
    unit: "money",
    currency: "USD",
    rows: [
      ...[0.628, 0.635, 0.645, 0.651, 0.583, 0.594].map((cost) => chartRow({ value: usd(cost) })),
      chartRow({ label: "T-118", value: usd("0.781"), marked: true }),
      chartRow({ label: "T-104", value: usd("0.742"), marked: true }),
    ],
    band: { from: null, to: null, low: "0.574", high: "0.702" },
    ref: { value: "0.638", label: "Fleet" },
  });
}

/** The Carriers frame's spans: a cover ending before one of its loads delivers. */
export function coverChart(): DeskBoardChart {
  return chart({
    unit: "count",
    ticks: [7, 14, 21, 30].map((days) => ({ at: NOW + days * DAY, label: "", marked: false })),
    rows: [
      chartRow({
        label: "Ridgeline Freight",
        spans: [{ from: NOW - 60 * DAY, to: NOW + 6 * DAY, kind: "cover" }],
        dots: [
          { at: NOW + DAY, value: null, marked: false, ghost: false, label: "" },
          {
            at: NOW + 8 * DAY,
            value: null,
            marked: true,
            ghost: false,
            label: "S26111040188306 delivers Dec 1",
          },
        ],
      }),
      chartRow({
        label: "Sandoval Trucking",
        spans: [{ from: NOW - 60 * DAY, to: NOW + 28 * DAY, kind: "cover" }],
      }),
    ],
  });
}
