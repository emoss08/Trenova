/**
 * A line break in a paragraph is a line break on screen. A model writes one
 * line under another expecting it to read that way, and markdown's rule that
 * a single newline is a space would run an address or a list of figures
 * into one line. The parser keeps a soft break as a newline inside the text
 * (and drops the two trailing spaces of a hard one), so each newline in a
 * text node becomes a break.
 *
 * Code spans are their own nodes and are never touched, and no break is put
 * beside a formula set on its own lines, which already starts a line.
 */

import type { InlineNode, MarkdownExtension } from "@tanstack/markdown";
import { isDisplayMath, isMath } from "./markdown-math";

function withBreaks(nodes: InlineNode[]): InlineNode[] {
  const out: InlineNode[] = [];
  nodes.forEach((node, index) => {
    if (node.type !== "text") {
      out.push(
        "children" in node && !isMath(node)
          ? ({ ...node, children: withBreaks(node.children) } as InlineNode)
          : node,
      );
      return;
    }
    if (!node.value.includes("\n")) {
      out.push(node);
      return;
    }
    const before = nodes[index - 1];
    const after = nodes[index + 1];
    let value = node.value;
    if (before && isDisplayMath(before)) value = value.replace(/^[ \t]*\n/u, "");
    if (after && isDisplayMath(after)) value = value.replace(/\n[ \t]*$/u, "");
    value.split(/[ \t]*\n[ \t]*/u).forEach((line, lineIndex) => {
      if (lineIndex > 0) out.push({ type: "break" });
      if (line !== "") out.push({ type: "text", value: line });
    });
  });

  return out;
}

export function breaksExtension(): MarkdownExtension {
  return {
    name: "breaks",
    transformInline: withBreaks,
  };
}
