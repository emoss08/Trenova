import { describe, expect, it } from "vitest";
import type { ExportColumn } from "../data-table-export";
import { buildPrintDocument } from "../data-table-print";

type Row = { name: string; amount: number };

const columns: ExportColumn<Row>[] = [
  { id: "name", header: "Name <b>", getValue: (row) => row.name },
  { id: "amount", header: "Amount", getValue: (row) => row.amount },
];

describe("buildPrintDocument", () => {
  it("escapes every title, header and value so no row is read as markup", () => {
    const html = buildPrintDocument({
      title: "Customers & <co>",
      subtitle: "2 rows",
      columns,
      rows: [{ name: '<img src=x onerror="alert(1)">', amount: 5 }],
    });

    expect(html).not.toContain("<img");
    expect(html).toContain("&lt;img src=x onerror=&quot;alert(1)&quot;&gt;");
    expect(html).toContain("<title>Customers &amp; &lt;co&gt;</title>");
    expect(html).toContain("<th>Name &lt;b&gt;</th>");
  });

  it("right-aligns numbers and keeps text left", () => {
    const html = buildPrintDocument({
      title: "T",
      subtitle: "",
      columns,
      rows: [{ name: "Acme", amount: 1200 }],
    });

    expect(html).toContain('<tr><td>Acme</td><td class="num">1200</td></tr>');
  });

  it("turns a wide table landscape and leaves a narrow one portrait", () => {
    const wide = Array.from({ length: 8 }, (_, index) => ({
      id: `c${index}`,
      header: `C${index}`,
      getValue: () => "",
    }));

    expect(buildPrintDocument({ title: "T", subtitle: "", columns: wide, rows: [] })).toContain(
      "size: landscape",
    );
    expect(buildPrintDocument({ title: "T", subtitle: "", columns, rows: [] })).not.toContain(
      "size: landscape",
    );
  });
});
