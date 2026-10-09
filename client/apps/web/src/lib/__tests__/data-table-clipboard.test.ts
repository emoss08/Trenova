import { afterEach, describe, expect, it, vi } from "vitest";
import type { ExportColumn } from "../data-table-export";
import { buildClipboardGrid, writeClipboardGrid } from "../data-table-clipboard";

type Row = { pro: string; customer: { name: string }; notes: string; amount: number };

const columns = new Map<string, ExportColumn<Row>>([
  ["pro", { id: "pro", header: "PRO", getValue: (row) => row.pro }],
  ["customer", { id: "customer", header: "Customer", getValue: (row) => row.customer.name }],
  ["notes", { id: "notes", header: "Notes", getValue: (row) => row.notes }],
  ["amount", { id: "amount", header: "Amount", getValue: (row) => row.amount }],
]);

const rows: Row[] = [
  { pro: "P-1", customer: { name: "Acme & Co" }, notes: "line one\nline two\tend", amount: 12.5 },
  { pro: "P-2", customer: { name: "<Big> \"Freight\"" }, notes: "", amount: 0 },
];

describe("buildClipboardGrid", () => {
  it("writes rows as tab-separated text in the order of the columns asked for", () => {
    const grid = buildClipboardGrid(rows, ["amount", "pro"], columns);

    expect(grid.text).toBe("12.5\tP-1\n0\tP-2");
  });

  it("keeps a value with a tab or line break in one cell", () => {
    const grid = buildClipboardGrid([rows[0]], ["notes"], columns);

    expect(grid.text).toBe("line one line two end");
  });

  it("adds the headers when asked", () => {
    const grid = buildClipboardGrid([rows[0]], ["pro", "customer"], columns, {
      includeHeader: true,
    });

    expect(grid.text).toBe("PRO\tCustomer\nP-1\tAcme & Co");
  });

  it("leaves out a column with no text form instead of copying a blank", () => {
    const grid = buildClipboardGrid([rows[0]], ["select", "pro"], columns);

    expect(grid.text).toBe("P-1");
  });

  it("escapes the HTML table a spreadsheet reads", () => {
    const grid = buildClipboardGrid([rows[1]], ["customer"], columns, { includeHeader: true });

    expect(grid.html).toBe(
      "<table><tr><th>Customer</th></tr><tr><td>&lt;Big&gt; &quot;Freight&quot;</td></tr></table>",
    );
  });
});

describe("writeClipboardGrid", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("falls back to plain text when the browser refuses HTML", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    const write = vi.fn().mockRejectedValue(new Error("not allowed"));
    vi.stubGlobal("navigator", { clipboard: { write, writeText } });
    vi.stubGlobal(
      "ClipboardItem",
      class {
        items: Record<string, Blob>;
        constructor(items: Record<string, Blob>) {
          this.items = items;
        }
      },
    );

    await writeClipboardGrid({ text: "a\tb", html: "<table></table>" });

    expect(write).toHaveBeenCalledTimes(1);
    expect(writeText).toHaveBeenCalledWith("a\tb");
  });
});
