// jsdom computes no layout: these hold structure, classes and words; that the keys stand on one
// line when wide and one to a line when narrow, and that the grid and table scroll sideways
// inside a narrow exhibit, needs a browser.
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BoardGrid } from "../../blocks/board-grid";
import { BoardLog } from "../../blocks/board-log";
import { BoardMetricTable } from "../../blocks/board-metric-table";
import { BoardSplit } from "../../blocks/board-split";
import type { DeskBoardGrid, DeskBoardSplit } from "../../blocks/metric-types";
import { count, NOW, row, usd, value } from "./fixtures";

afterEach(cleanup);

const BATCH: DeskBoardSplit = {
  total: usd("144780.00"),
  parts: [
    { label: "Ready", value: usd("121952.10"), kind: "used" },
    { label: "Held", value: usd("1412.50"), kind: "over" },
    { label: "Deducted", value: usd("21415.40"), kind: "credit" },
  ],
};

describe("BoardSplit", () => {
  it("draws the headline figure, the strip in its parts and each part keyed with its value", () => {
    const { container } = render(
      <BoardSplit
        split={BATCH}
        figures={[
          { name: "net", label: "net", value: usd("123364.60"), was: null, wasAt: null, tone: "" },
        ]}
      />,
    );
    expect(container.querySelector(".dk-bd-big")?.textContent).toBe("$123,364.60net");
    const strip = screen.getByRole("img");
    expect(strip.getAttribute("aria-label")).toBe(
      "Of $144,780.00: Ready $121,952.10, Held $1,412.50, Deducted $21,415.40",
    );
    const fills = [...strip.querySelectorAll("i")].map((fill) => ({
      className: fill.className,
      width: (fill as HTMLElement).style.width,
    }));
    expect(fills.map((fill) => fill.className)).toEqual([
      "dk-bd-used",
      "dk-bd-over",
      "dk-bd-credit",
    ]);
    expect(Number.parseFloat(fills[0].width)).toBeCloseTo(84.23, 1);
    expect(
      [...container.querySelectorAll(".dk-bd-keys b")].map((node) => node.textContent),
    ).toEqual(["$121,952.10", "$1,412.50", "$21,415.40"]);
  });

  it("falls back to the whole when no figure was read, and draws no fill for an empty part", () => {
    const { container } = render(
      <BoardSplit
        split={{
          ...BATCH,
          parts: [...BATCH.parts, { label: "Advanced", value: usd(0), kind: "used" }],
        }}
        figures={[{ name: "net", label: "net", value: null, was: null, wasAt: null, tone: "" }]}
      />,
    );
    expect(container.querySelector(".dk-bd-big")?.textContent).toBe("$144,780.00");
    expect(screen.getByRole("img").querySelectorAll("i")).toHaveLength(3);
    expect(container.querySelectorAll(".dk-bd-keys > span")).toHaveLength(4);
  });
});

const COVER: DeskBoardGrid = {
  columns: [
    { label: "Mon", sublabel: "Nov 30" },
    { label: "Tue", sublabel: "Dec 1" },
    { label: "Thu", sublabel: "Dec 3" },
  ],
  rows: [
    {
      label: "Dallas–Fort Worth",
      cells: [
        { have: 22, need: 20, needHigh: 20, state: "covered" },
        { have: 19, need: 23, needHigh: 23, state: "deep" },
        { have: 20, need: 22, needHigh: 22, state: "short" },
      ],
    },
    {
      label: "Atlanta",
      cells: [
        { have: 11, need: 10, needHigh: 10, state: "covered" },
        { have: 11, need: 11, needHigh: 11, state: "covered" },
        { have: 10, need: 10, needHigh: 12, state: "maybe" },
      ],
    },
  ],
};

describe("BoardGrid", () => {
  it("writes each cell's balance, hatches the short, outlines the deepest and the uncertain", () => {
    const { container } = render(<BoardGrid grid={COVER} title="Drivers against need" />);
    expect(screen.getByRole("img").getAttribute("aria-label")).toBe(
      "Drivers against need: 2 of 6 short, 1 may be short; the worst 4 short, Dallas–Fort Worth at Tue",
    );
    const cells = [...container.querySelectorAll(".gc")].map((cell) => ({
      className: cell.className,
      head: cell.querySelector("b")?.textContent,
      foot: cell.querySelector("small")?.textContent,
    }));
    expect(cells).toEqual([
      { className: "gc", head: "+2", foot: "22/20" },
      { className: "gc short deep", head: "−4", foot: "19/23" },
      { className: "gc short", head: "−2", foot: "20/22" },
      { className: "gc", head: "+1", foot: "11/10" },
      { className: "gc", head: "0", foot: "11/11" },
      { className: "gc maybe", head: "0 to −2", foot: "10/10–12" },
    ]);
    expect(container.querySelectorAll(".gh-h")).toHaveLength(4);
    const grid = container.querySelector<HTMLElement>(".dk-bd-grid");
    expect(grid?.style.getPropertyValue("--n")).toBe("3");
    expect(container.querySelector(".dk-grid")).toBeNull();
    expect(container.querySelectorAll(".gh-a")).toHaveLength(2);
  });

  it("says every cell is covered when none is short", () => {
    render(
      <BoardGrid
        grid={{ ...COVER, rows: [{ ...COVER.rows[1], cells: COVER.rows[1].cells.slice(0, 2) }] }}
        title="Cover"
      />,
    );
    expect(screen.getByRole("img").getAttribute("aria-label")).toBe("Cover: all 2 cells covered");
  });
});

describe("BoardLog", () => {
  it("draws each entry with its time, tone and detail, and opens the records it names", () => {
    const open = vi.fn();
    const shipment = row("shipment", "shp_1", "S26110041402218");
    const { container } = render(
      <BoardLog
        entries={[
          {
            at: NOW - 3600,
            tone: "",
            text: "Tender 88214 became S26110041402218",
            detail: "204 in, 990 out",
            records: [shipment],
          },
          {
            at: NOW - 1800,
            tone: "warn",
            text: "Tender 88231 was quarantined",
            detail: "",
            records: [],
          },
        ]}
        now={NOW}
        open={open}
        opensInPlace={() => true}
      />,
    );
    const items = container.querySelectorAll("li");
    expect(items).toHaveLength(2);
    expect(items[1].className).toBe("dk-warn");
    expect(items[0].querySelector("div > span")?.getAttribute("title")).toBe(
      "Tender 88214 became S26110041402218",
    );
    expect(items[1].querySelector("em")).toBeNull();
    screen.getByRole("button", { name: "S26110041402218" }).click();
    expect(open).toHaveBeenCalledWith(shipment, ["shp_1"]);
  });
});

describe("BoardLog times", () => {
  const entry = (at: number) => ({ at, tone: "", text: "Tender in", detail: "", records: [] });
  const midday = new Date(NOW * 1000);
  midday.setHours(12, 0, 0, 0);
  const noon = Math.floor(midday.getTime() / 1000);

  it("writes the time alone when every entry is today, and the day with it otherwise", () => {
    const { container } = render(
      <BoardLog
        entries={[entry(noon - 3600), entry(noon - 2 * 86_400)]}
        now={noon}
        open={vi.fn()}
        opensInPlace={() => true}
      />,
    );
    const times = [...container.querySelectorAll("time")].map((node) => node.textContent ?? "");
    expect(times[0]).not.toMatch(/,/);
    expect(times[1]).toMatch(/,/);
    expect(container.querySelector("ol")?.className).toBe("dk-bd-log dk-bd-mlog");
  });
});

describe("BoardMetricTable", () => {
  it("names each row, sets values in their cells and marks hot, off and the crossing cell", () => {
    const { container } = render(
      <BoardMetricTable
        title="The week"
        table={{
          columns: [
            { label: "Person", kind: "text" },
            { label: "Sun 22", kind: "value" },
            { label: "Mon 23", kind: "value" },
            { label: "Week", kind: "value" },
          ],
          rows: [
            [
              { value: null, text: "Hector Alvarez", mark: "" },
              { value: value("duration", 41400), text: "", mark: "hot" },
              { value: null, text: "", mark: "off" },
              { value: value("duration", 174600), text: "", mark: "w" },
            ],
            [
              { value: null, text: "Rosa Lind", mark: "" },
              { value: count(0), text: "", mark: "" },
            ],
          ],
        }}
      />,
    );
    const cells = [...container.querySelectorAll("tbody tr")].map((tr) =>
      [...tr.querySelectorAll("td")].map((td) => `${td.className}|${td.textContent}`),
    );
    expect(cells).toEqual([
      ["dk-txt|Hector Alvarez", "hot|11h 30m", "off|–", "w|48h 30m"],
      ["dk-txt|Rosa Lind", "|0", "off|–", "off|–"],
    ]);
    expect(screen.getByRole("table", { name: "The week" })).toBeTruthy();
  });
});
