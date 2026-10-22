import {
  formatReportColumns,
  reportCellText,
  resolveReportDisplays,
  type ReportCell,
  type ReportFormatColumn,
  type ReportResolveInput,
} from "@trenova/shared/rules/report";
import { rulesSync } from "@trenova/shared/rules/runtime";
import {
  corpusCase,
  corpusMismatches,
  expectCorpusOutput,
} from "@trenova/shared/rules/__tests__/corpus";
import { describe, expect, it } from "vitest";

type WireCell = { text?: string; date?: string; tone?: ReportCell["tone"] };

/** A corpus cell as the wrapper hands it back: a text or a date, always with a tone. */
function asCell(cell: WireCell): ReportCell {
  const tone = cell.tone ?? "";
  return cell.date === undefined ? { text: cell.text ?? "", tone } : { date: cell.date, tone };
}

function formatCase(name: string) {
  const testCase = corpusCase("report", name);
  const input = testCase.input as { columns: ReportFormatColumn[] };
  const output = testCase.output as { columns: { values: WireCell[] }[] };
  expect(formatReportColumns(rulesSync(), input.columns)).toEqual(
    output.columns.map((column) => column.values.map(asCell)),
  );
}

describe("formatReportColumns", () => {
  it("rounds half away from zero on the digits as written: 1.005 is 1.01", () => {
    formatCase("rounding is StringFixed's half away from zero, on the digits as written");
  });

  it("reads decimal strings as numbers in a decimal column", () => {
    formatCase("a decimal string in a decimal column rounds exactly too");
  });

  it("hands a number in a date column back as whole epoch seconds", () => {
    formatCase("reportfmt: a number in a date column is a date for the host to write");
  });

  it("tones a value by the first rule it matches", () => {
    formatCase("reportfmt: the first matching tone rule wins");
  });

  it("renders a banded value as the range it stands for", () => {
    formatCase("TS: banded columns");
  });

  it("renders an undefined cell of a column without a display as empty", () => {
    const [[cell]] = formatReportColumns(rulesSync(), [{ display: null, values: [undefined] }]);
    expect(cell).toEqual({ text: "", tone: "" });
  });
});

describe("resolveReportDisplays", () => {
  it("carries bands and rules with exact decimal strings", () => {
    const testCase = corpusCase(
      "report",
      "resolve: bands and rules are carried; an empty band is dropped, a malformed one kept for validation",
    );
    const input = testCase.input as { columns: ReportResolveInput[] };
    expect(
      corpusMismatches(testCase.output, {
        displays: resolveReportDisplays(rulesSync(), input.columns),
      }),
    ).toEqual([]);
  });

  it("derives a column's display from its type and hint", () => {
    const testCase = corpusCase("report", "resolve: defaults from type and format hint");
    const input = testCase.input as { columns: ReportResolveInput[] };
    expectCorpusOutput(testCase, {
      displays: resolveReportDisplays(rulesSync(), input.columns),
    });
  });
});

describe("reportCellText", () => {
  it("writes a date cell in the report's time zone and the column's date style", () => {
    expect(reportCellText({ date: "1772587800", tone: "" }, { dateStyle: "date" }, "UTC")).toBe(
      "2026-03-04",
    );
    expect(reportCellText({ text: "$5.00", tone: "warning" }, { dateStyle: "date" })).toBe("$5.00");
  });
});
