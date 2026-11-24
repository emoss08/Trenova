/**
 * Math in a reply: `$…$` inline and `$$…$$` on lines of their own, set as a
 * formula by the typesetter once it has been read. Each formula becomes a
 * component node whose tag the renderer maps to the element that draws it,
 * carrying the TeX as written.
 *
 * An escaped `\$` is never math: the parser reads escapes before any
 * extension sees the text, which is what keeps the money prepareMarkdown
 * escaped as money. Code spans and fences are left as written for the same
 * reason.
 */

import type {
  BlockNode,
  BlockParseContext,
  InlineComponentNode,
  InlineNode,
  InlineParseContext,
  MarkdownExtension,
} from "@tanstack/markdown";

/** The tag a formula is emitted as; ai-markdown maps it to the component that draws it. */
export const MATH_TAG = "md-math";

/** How a formula sits: in a line of text, alone on its lines, or as a block of its own. */
export type MathMode = "inline" | "display" | "block";

function formula(tex: string, mode: MathMode) {
  return {
    name: "math",
    tagName: MATH_TAG,
    attributes: { tex, mode },
    properties: { "data-tex": tex, "data-mode": mode },
  };
}

/** Whether a node is a formula, which no other extension looks inside. */
export function isMath(node: BlockNode | InlineNode): boolean {
  return (node.type === "component" || node.type === "inlineComponent") && node.name === "math";
}

/** Whether an inline node is a formula set on lines of its own. */
export function isDisplayMath(node: InlineNode): boolean {
  return isMath(node) && (node as InlineComponentNode).attributes.mode === "display";
}

/** Where the run of `size` dollars that closes a formula starts, skipping escaped ones; -1 if none. */
function closingRun(source: string, from: number, size: number): number {
  for (let index = from; index < source.length; index += 1) {
    const char = source[index];
    if (char === "\\") {
      index += 1;
      continue;
    }
    if (char !== "$") continue;
    let run = 1;
    while (source[index + run] === "$") run += 1;
    if (run === size) return index;
    index += run - 1;
  }

  return -1;
}

function parseInlineMath({ source, index }: InlineParseContext) {
  const size = source[index + 1] === "$" ? 2 : 1;
  if (source[index + size] === "$") return undefined;
  const close = closingRun(source, index + size, size);
  if (close === -1) return undefined;
  const tex = source.slice(index + size, close).trim();
  if (tex === "") return undefined;
  const end = close + size;
  // `$$…$$` that has a line to itself inside a paragraph is a displayed
  // formula, as it would be had a blank line come before it.
  const display =
    size === 2 &&
    (index === 0 || source[index - 1] === "\n") &&
    (end === source.length || source[end] === "\n");
  const node: InlineComponentNode = {
    type: "inlineComponent",
    ...formula(tex, display ? "display" : "inline"),
    children: [],
  };

  return { node, length: end - index };
}

const OPEN = /^ {0,3}\$\$(.*)$/u;
const CLOSE = /^(.*?)\$\$\s*$/u;

/**
 * A `$$` block: from a line that opens with `$$` to the line that ends with
 * one, or to the end of the reply when it never closes. A line that opens
 * and closes and has text after is a paragraph, left to the inline parser.
 */
function parseMathBlock(context: BlockParseContext): BlockNode | undefined {
  const open = OPEN.exec(context.lines[context.index] ?? "");
  if (!open) return undefined;
  const rest = open[1];
  const lines: string[] = [];
  let cursor = context.index + 1;
  if (rest.includes("$$")) {
    const single = CLOSE.exec(rest);
    if (!single || single[1].includes("$$")) return undefined;
    lines.push(single[1]);
  } else {
    lines.push(rest);
    while (cursor < context.lines.length) {
      const line = context.lines[cursor] ?? "";
      cursor += 1;
      const close = CLOSE.exec(line);
      if (close) {
        lines.push(close[1]);
        break;
      }
      lines.push(line);
    }
  }
  const tex = lines.join("\n").trim();
  if (tex === "") return undefined;
  context.consume(cursor - context.index);

  return { type: "component", ...formula(tex, "block"), children: [] };
}

export function mathExtension(): MarkdownExtension {
  return {
    name: "math",
    parseBlock: parseMathBlock,
    inlineParser: { markers: "$", parse: parseInlineMath },
  };
}
