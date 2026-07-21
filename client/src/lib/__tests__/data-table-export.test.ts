import { describe, expect, it } from "vitest";
import { buildCsv, type ExportColumn } from "../data-table-export";

type TestRow = {
  id: string;
  name: string;
  amount: number | null;
  tags: string[];
  worker?: { firstName: string };
};

const columns: ExportColumn<TestRow>[] = [
  { id: "id", header: "ID", getValue: (row) => row.id },
  { id: "name", header: "Name", getValue: (row) => row.name },
  { id: "amount", header: "Amount, USD", getValue: (row) => row.amount },
  { id: "tags", header: "Tags", getValue: (row) => row.tags },
  { id: "worker", header: "Worker", getValue: (row) => row.worker?.firstName },
];

describe("buildCsv", () => {
  it("emits a header row followed by data rows", () => {
    const csv = buildCsv<TestRow>(
      [{ id: "1", name: "Alpha", amount: 10, tags: [], worker: { firstName: "Ann" } }],
      columns,
    );
    const lines = csv.split("\r\n");

    expect(lines).toHaveLength(2);
    expect(lines[0]).toBe('ID,Name,"Amount, USD",Tags,Worker');
    expect(lines[1]).toBe("1,Alpha,10,[],Ann");
  });

  it("escapes quotes, commas, and newlines", () => {
    const csv = buildCsv<TestRow>(
      [{ id: "1", name: 'He said "hi",\nthen left', amount: null, tags: [] }],
      columns,
    );
    const lines = csv.split("\r\n");

    expect(lines[1]).toBe('1,"He said ""hi"",\nthen left",,[],');
  });

  it("serializes arrays and objects as JSON", () => {
    const csv = buildCsv<TestRow>(
      [{ id: "1", name: "A", amount: 0, tags: ["x", "y"] }],
      columns,
    );

    expect(csv.split("\r\n")[1]).toBe('1,A,0,"[""x"",""y""]",');
  });

  it("handles empty row sets", () => {
    const csv = buildCsv<TestRow>([], columns);
    expect(csv).toBe('ID,Name,"Amount, USD",Tags,Worker');
  });
});
