/**
 * For a reply still arriving: every word becomes its own span with the given
 * class, so a stylesheet can bring each one in as it lands, and the last
 * block can end in a caret. A word already on screen keeps its element
 * across re-renders and does not animate again; only the new ones mount.
 * Code and formulas are left whole.
 */

import type {
  BlockNode,
  InlineComponentNode,
  InlineNode,
  MarkdownDocument,
  MarkdownExtension,
} from "@tanstack/markdown";
import { isMath } from "./markdown-math";

function span(className: string, children: InlineNode[], ariaHidden = false): InlineComponentNode {
  return {
    type: "inlineComponent",
    name: "stream-word",
    tagName: "span",
    attributes: {},
    properties: ariaHidden ? { className, "aria-hidden": "true" } : { className },
    children,
  };
}

function wordsOf(text: string, className: string): InlineNode[] {
  // Each word keeps the space after it, so wrapping never changes the text.
  return text
    .split(/(?<=\s)(?=\S)/u)
    .map((part) =>
      /\S/u.test(part)
        ? span(className, [{ type: "text", value: part }])
        : { type: "text", value: part },
    );
}

function wrapInline(nodes: InlineNode[], className: string): InlineNode[] {
  return nodes.flatMap((node): InlineNode[] => {
    if (node.type === "text") return wordsOf(node.value, className);
    if ("children" in node && !isMath(node)) {
      return [{ ...node, children: wrapInline(node.children, className) } as InlineNode];
    }
    return [node];
  });
}

function wrapBlocks(blocks: BlockNode[], className: string): BlockNode[] {
  return blocks.map((block): BlockNode => {
    switch (block.type) {
      case "heading":
      case "paragraph":
        return { ...block, children: wrapInline(block.children, className) };
      case "blockquote":
      case "callout":
        return { ...block, children: wrapBlocks(block.children, className) };
      case "component":
        return isMath(block)
          ? block
          : { ...block, children: wrapBlocks(block.children, className) };
      case "list":
        return {
          ...block,
          items: block.items.map((item) => ({
            ...item,
            children: wrapBlocks(item.children, className),
          })),
        };
      case "table": {
        const cell = (value: (typeof block.header)[number]) => ({
          ...value,
          children: wrapInline(value.children, className),
        });
        return {
          ...block,
          header: block.header.map(cell),
          rows: block.rows.map((row) => row.map(cell)),
        };
      }
      case "footnotes":
        return {
          ...block,
          items: block.items.map((item) => ({
            ...item,
            children: wrapBlocks(item.children, className),
          })),
        };
      default:
        return block;
    }
  });
}

/** Whether a run of inline nodes holds a word outside code and formulas. */
function hasWords(nodes: InlineNode[]): boolean {
  return nodes.some((node) =>
    node.type === "text"
      ? /\S/u.test(node.value)
      : "children" in node && !isMath(node) && hasWords(node.children),
  );
}

/**
 * The run of inline nodes the reply's last words sit in, where the caret
 * goes after them: the last block that has words, read from the end, past
 * any code or formula that closes the reply.
 */
function lastWords(blocks: BlockNode[]): InlineNode[] | null {
  for (let index = blocks.length - 1; index >= 0; index -= 1) {
    const block = blocks[index];
    let found: InlineNode[] | null = null;
    switch (block.type) {
      case "heading":
      case "paragraph":
        found = hasWords(block.children) ? block.children : null;
        break;
      case "blockquote":
      case "callout":
        found = lastWords(block.children);
        break;
      case "component":
        found = isMath(block) ? null : lastWords(block.children);
        break;
      case "list":
        for (let item = block.items.length - 1; item >= 0 && !found; item -= 1) {
          found = lastWords(block.items[item].children);
        }
        break;
      case "table":
        for (const row of [...block.rows].reverse().concat([block.header])) {
          const cell = [...row].reverse().find((value) => hasWords(value.children));
          if (cell) {
            found = cell.children;
            break;
          }
        }
        break;
      default:
        break;
    }
    if (found) return found;
  }

  return null;
}

export function streamWordsExtension(options: {
  className: string;
  caret?: string;
}): MarkdownExtension {
  return {
    name: "stream-words",
    transformDocument: (document: MarkdownDocument) => {
      const children = options.className
        ? wrapBlocks(document.children, options.className)
        : document.children;
      if (options.caret) {
        lastWords(children)?.push(span(options.caret, [], true));
      }
      return { ...document, children };
    },
  };
}
