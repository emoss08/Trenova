/**
 * A rehype plugin that right-aligns a table's number columns, the way any
 * table of figures is set: a column whose every filled body cell is a number
 * (an amount, a count, a percentage, a change like "+18%") lines up on the
 * right and takes the tabular class. A column the author aligned keeps the
 * alignment they wrote.
 */

type HastNode = {
  type: string;
  tagName?: string;
  value?: string;
  properties?: Record<string, unknown>;
  children?: HastNode[];
};

/** A cell that reads as a figure, or as the dash a table writes for none. */
const FIGURE =
  /^[(+\-−]?\s*[$€£¥]?\s*\d[\d,]*(\.\d+)?\s*(%|[kKmMbB]|h|hrs?|mi|lb|lbs|kg|x|×)?\)?$/u;
const NONE = /^[-–—]?$/u;

function textOf(node: HastNode): string {
  if (node.type === "text") return node.value ?? "";
  return (node.children ?? []).map(textOf).join("");
}

function elements(node: HastNode, tagName: string): HastNode[] {
  const found: HastNode[] = [];
  for (const child of node.children ?? []) {
    if (child.type !== "element") continue;
    if (child.tagName === tagName) found.push(child);
    else found.push(...elements(child, tagName));
  }
  return found;
}

function cellsOf(row: HastNode): HastNode[] {
  return (row.children ?? []).filter(
    (child) => child.type === "element" && (child.tagName === "td" || child.tagName === "th"),
  );
}

function addClass(node: HastNode, className: string) {
  const properties = (node.properties ??= {});
  const current = properties.className;
  const list = Array.isArray(current) ? current : typeof current === "string" ? [current] : [];
  properties.className = [...list, className];
}

function alignTable(table: HastNode, className: string) {
  const rows = elements(table, "tr");
  const head = rows.filter((row) => cellsOf(row).some((cell) => cell.tagName === "th"));
  const body = rows.filter((row) => !head.includes(row));
  if (body.length === 0) return;

  const columns = Math.max(...rows.map((row) => cellsOf(row).length));
  for (let column = 0; column < columns; column += 1) {
    const cells = rows.map((row) => cellsOf(row)[column]).filter(Boolean);
    if (cells.some((cell) => cell.properties?.align)) continue;
    const filled = body
      .map((row) => cellsOf(row)[column])
      .filter((cell): cell is HastNode => Boolean(cell))
      .map((cell) => textOf(cell).trim())
      .filter((text) => !NONE.test(text));
    if (filled.length === 0 || !filled.every((text) => FIGURE.test(text))) continue;
    for (const cell of cells) {
      (cell.properties ??= {}).align = "right";
      addClass(cell, className);
    }
  }
}

export function rehypeNumericColumns(options: { className: string }) {
  return (tree: HastNode) => {
    for (const table of elements(tree, "table")) {
      alignTable(table, options.className);
    }
  };
}
