/**
 * Right-aligns a table's number columns, the way any table of figures is
 * set: a column whose every filled body cell is a number (an amount, a count,
 * a percentage, a change like "+18%") lines up on the right and is set as
 * figures. A column the author aligned keeps the alignment they wrote.
 *
 * Alignment alone is not enough: the stylesheet sets a figure cell in the
 * mono face with tabular digits and never wraps it (`td.md-num`), and the
 * parser gives a cell no attributes of its own. So the column's alignment is
 * set on the table, and each of its cells' content is wrapped in a marker
 * node the cell component finds and turns into the class.
 */

import type {
  BlockNode,
  InlineComponentNode,
  InlineNode,
  MarkdownDocument,
  MarkdownExtension,
  TableCellNode,
  TableNode,
} from "@tanstack/markdown";

/** The tag a figure cell's content is wrapped in; see FIGURE_CELL_PROPERTY. */
export const FIGURE_CELL_TAG = "md-num-cell";

/** The property on the marker that tells a cell it holds a figure. */
export const FIGURE_CELL_PROPERTY = "data-md-num";

/** A cell that reads as a figure, or as the dash a table writes for none. */
const FIGURE =
  /^[(+\-−]?\s*[$€£¥]?\s*\d[\d,]*(\.\d+)?\s*(%|[kKmMbB]|h|hrs?|mi|lb|lbs|kg|x|×)?\)?$/u;
const NONE = /^[-–—]?$/u;

function textOf(nodes: InlineNode[]): string {
  return nodes
    .map((node) => {
      if (node.type === "text" || node.type === "inlineCode") return node.value;
      if ("children" in node) return textOf(node.children);
      return "";
    })
    .join("");
}

function marked(cell: TableCellNode): TableCellNode {
  const marker: InlineComponentNode = {
    type: "inlineComponent",
    name: "figure-cell",
    tagName: FIGURE_CELL_TAG,
    attributes: {},
    properties: { [FIGURE_CELL_PROPERTY]: "" },
    children: cell.children,
  };
  return { ...cell, children: [marker] };
}

function alignTable(table: TableNode): TableNode {
  if (table.rows.length === 0) return table;
  const figures = table.header.map((_, column) => {
    if (table.align[column]) return false;
    const filled = table.rows
      .map((row) => (row[column] ? textOf(row[column].children).trim() : ""))
      .filter((text) => !NONE.test(text));
    return filled.length > 0 && filled.every((text) => FIGURE.test(text));
  });
  if (!figures.includes(true)) return table;
  const mark = (cell: TableCellNode, column: number) => (figures[column] ? marked(cell) : cell);

  return {
    ...table,
    align: table.align.map((align, column) => (figures[column] ? "right" : align)),
    header: table.header.map(mark),
    rows: table.rows.map((row) => row.map(mark)),
  };
}

function alignBlocks(blocks: BlockNode[]): BlockNode[] {
  return blocks.map((block) => {
    switch (block.type) {
      case "table":
        return alignTable(block);
      case "blockquote":
      case "callout":
      case "component":
        return { ...block, children: alignBlocks(block.children) };
      case "list":
        return {
          ...block,
          items: block.items.map((item) => ({ ...item, children: alignBlocks(item.children) })),
        };
      default:
        return block;
    }
  });
}

export function numericColumnsExtension(): MarkdownExtension {
  return {
    name: "numeric-columns",
    transformDocument: (document: MarkdownDocument) => ({
      ...document,
      children: alignBlocks(document.children),
    }),
  };
}
