/**
 * Keeps a reply's markdown to the set the Desk draws. Markdown brings a few
 * things the design leaves out on purpose: a bare address turning into a
 * link, footnotes and images. Each is put back as the words that were
 * written, so the reply reads as typed rather than as something the reader
 * did not expect.
 *
 * - A bare address stays text: the Desk leaves the autolinks extension off
 *   and reads only an address written in angle brackets, `<https://…>`, as a
 *   link, since the author asked for that one.
 * - An image is written back as `![alt](src)` from its node, since the parser
 *   keeps no source positions.
 * - A footnote never forms: its definition is escaped before parsing
 *   (prepareMarkdown's `deskSubset`), because the parser lifts definitions
 *   out of the text before any extension runs and drops the ones nothing
 *   refers to. Unescaped, `[^1]` alone is not a reference and stays text.
 * - Code indented rather than fenced is already prose: the parser has no
 *   indented code.
 */

import type { BlockNode, InlineNode, MarkdownExtension } from "@tanstack/markdown";
import { autolinksExtension } from "@tanstack/markdown/extensions/autolinks";

function written(node: Extract<InlineNode, { type: "image" }>): string {
  const title = node.title ? ` "${node.title}"` : "";
  return `![${node.alt}](${node.src}${title})`;
}

function inline(nodes: InlineNode[]): InlineNode[] {
  return nodes.map((node): InlineNode => {
    if (node.type === "image") return { type: "text", value: written(node) };
    if ("children" in node) return { ...node, children: inline(node.children) } as InlineNode;
    return node;
  });
}

function blocks(nodes: BlockNode[]): BlockNode[] {
  return nodes.map((block): BlockNode => {
    switch (block.type) {
      case "heading":
      case "paragraph":
        return { ...block, children: inline(block.children) };
      case "blockquote":
      case "callout":
      case "component":
        return { ...block, children: blocks(block.children) };
      case "list":
        return {
          ...block,
          items: block.items.map((item) => ({ ...item, children: blocks(item.children) })),
        };
      case "table": {
        const cell = (value: (typeof block.header)[number]) => ({
          ...value,
          children: inline(value.children),
        });
        return {
          ...block,
          header: block.header.map(cell),
          rows: block.rows.map((row) => row.map(cell)),
        };
      }
      default:
        return block;
    }
  });
}

const angleLinks = autolinksExtension().inlineParser;

export function deskSubsetExtension(): MarkdownExtension {
  return {
    name: "desk-subset",
    ...(angleLinks && { inlineParser: { markers: "<", parse: angleLinks.parse } }),
    transformDocument: (document) => ({ ...document, children: blocks(document.children) }),
  };
}
