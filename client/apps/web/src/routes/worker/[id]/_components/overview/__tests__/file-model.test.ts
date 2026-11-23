import type { WorkerArea } from "@/routes/worker/use-worker-state";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describe, expect, it } from "vitest";
import { payRateText } from "../../overview-model";
import { fileCellRules, fileCells, fileColumns } from "../file-model";
import { overviewFixture } from "./overview-fixtures";

const t = translate;
const everything = { isDrawable: () => true };

describe("fileCells", () => {
  it("says where each part of the record stands, in the file's order", () => {
    const cells = fileCells(overviewFixture(), everything, t);
    expect(
      cells.map((cell) => [
        cell.label,
        cell.state,
        cell.note?.text ?? null,
        cell.note?.tone ?? null,
      ]),
    ).toEqual([
      ["Credentials", "8 of 8 required on file", "1 expiring", "warning"],
      ["DQ file", "Incomplete", "1 outstanding", "warning"],
      ["Drug and alcohol", "Clear", expect.stringMatching(/^Query due /), "warning"],
      ["Training", "5 of 6 current", "1 due soon", "warning"],
      ["Safety", "Score 74 · Watch", "7 active points", "warning"],
      [
        "Reviews",
        "Last: 4.2",
        `Next due ${formatUnixDateMedium(Date.UTC(2027, 0, 15, 12) / 1000)}`,
        "muted",
      ],
      ["Checklists", "Annual file review 2026", "5 of 8 settled, 1 overdue", "danger"],
      ["Time off", "Vacation: 9.5 days", "2 days requested", "muted"],
      ["Leave", "No open case", "480 of 480 hours left", "muted"],
      ["Pay", "OTR mileage", "$0.62 a loaded mile", "muted"],
      ["Benefits", "2 plans covered", "$200.90 a period", "muted"],
      ["Driver portal", "Linked", "1 change waiting", "warning"],
    ]);
  });

  it("opens each cell's tab and view", () => {
    const cells = fileCells(overviewFixture(), everything, t);
    expect(cells.find((cell) => cell.key === "safety")?.area).toEqual({
      tab: "safety",
      view: null,
    });
    expect(cells.find((cell) => cell.key === "benefits")?.area).toEqual({
      tab: "pay",
      view: "benefits",
    });
  });

  it("leaves out the parts a tenant's plan or the reader's permissions leave out", () => {
    const hidden = new Set(["safety", "pay-profile", "benefits"]);
    const areas = {
      isDrawable: (area: WorkerArea) => !hidden.has(area.view ?? area.tab),
    };
    const keys = fileCells(overviewFixture(), areas, t).map((cell) => cell.key);
    expect(keys).toEqual([
      "credentials",
      "dq-file",
      "drug-and-alcohol",
      "training",
      "reviews",
      "checklists",
      "time-off",
      "leave",
      "driver-portal",
    ]);
  });

  it("draws a section the read withheld as withheld, never as empty", () => {
    const cells = fileCells(
      overviewFixture({ credentials: null, portal: null, pto: null }),
      everything,
      t,
    );
    for (const key of ["credentials", "driver-portal", "time-off"]) {
      expect(cells.find((cell) => cell.key === key)).toMatchObject({ state: null, note: null });
    }
  });

  it("says when no pay profile is in force rather than withholding the cell", () => {
    const pay = fileCells(overviewFixture({ pay: null }), everything, t).find(
      (cell) => cell.key === "pay-profile",
    );
    expect(pay).toMatchObject({ state: "No pay profile", note: { tone: "warning" } });
  });
});

describe("payRateText", () => {
  it("quotes the rate by how it is calculated", () => {
    expect(payRateText("PerLoadedMile", "0.62", t)).toBe("$0.62 a loaded mile");
    expect(payRateText("PerHour", "28", t)).toBe("$28.00 an hour");
    expect(payRateText("PercentOfRevenue", "27.5", t)).toBe("27.5% of revenue");
    expect(payRateText("PerTotalMile", "0.585", t)).toBe("$0.585 a mile");
  });
});

describe("fileColumns and fileCellRules", () => {
  it("takes three columns, then two, then one as the width narrows", () => {
    expect([1200, 720, 719, 440, 439, 0].map(fileColumns)).toEqual([3, 3, 2, 2, 1, 3]);
  });

  it("rules a full grid between cells only", () => {
    const rules = Array.from({ length: 12 }, (_, index) => fileCellRules(index, 12, 3));
    expect(rules.map((rule) => rule.right)).toEqual([
      true,
      true,
      false,
      true,
      true,
      false,
      true,
      true,
      false,
      true,
      true,
      false,
    ]);
    expect(rules.map((rule) => rule.bottom)).toEqual([
      true,
      true,
      true,
      true,
      true,
      true,
      true,
      true,
      true,
      false,
      false,
      false,
    ]);
  });

  it("closes a partial last row without a rule past its last cell", () => {
    const rules = Array.from({ length: 5 }, (_, index) => fileCellRules(index, 5, 3));
    expect(rules).toEqual([
      { right: true, bottom: true },
      { right: true, bottom: true },
      { right: false, bottom: true },
      { right: true, bottom: false },
      { right: false, bottom: false },
    ]);
  });

  it("draws only row rules in one column and none for a single cell", () => {
    expect(Array.from({ length: 3 }, (_, index) => fileCellRules(index, 3, 1))).toEqual([
      { right: false, bottom: true },
      { right: false, bottom: true },
      { right: false, bottom: false },
    ]);
    expect(fileCellRules(0, 1, 3)).toEqual({ right: false, bottom: false });
  });
});
