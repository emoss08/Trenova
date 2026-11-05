import { describe, expect, it } from "vitest";
import { buildCsv } from "../data-table-export";

type Row = { text: string; amount: number; shown: string };

describe("a CSV of stored text", () => {
  const columns = [
    { header: "Text", getValue: (row: Row) => row.text },
    { header: "Amount", getValue: (row: Row) => row.amount },
    { header: "Shown", getValue: (row: Row) => row.shown },
  ];

  it("never hands a spreadsheet a formula", () => {
    const csv = buildCsv(
      [{ text: '=HYPERLINK("https://x.test")', amount: -5, shown: "-12.50" }],
      columns as never,
    );

    expect(csv.split("\r\n")[1]).toBe(`"'=HYPERLINK(""https://x.test"")",-5,-12.50`);
  });
});
