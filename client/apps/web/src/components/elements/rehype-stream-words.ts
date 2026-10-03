/**
 * A rehype plugin for a reply still arriving: every word becomes its own
 * span with the given class, so a stylesheet can bring each one in as it
 * lands, and the last block can end in a caret. A word already on screen
 * keeps its element across re-renders and does not animate again; only the
 * new ones mount. Code is left as written.
 */

type HastNode = {
  type: string;
  tagName?: string;
  value?: string;
  properties?: Record<string, unknown>;
  children?: HastNode[];
};

const SKIP = new Set(["code", "pre", "math", "svg"]);

/** Typeset math is drawn by KaTeX and left whole. */
function isMath(node: HastNode): boolean {
  const className = node.properties?.className;
  const list = Array.isArray(className) ? className : [];
  return list.some((name) => typeof name === "string" && name.startsWith("katex"));
}

function wordsOf(text: string, className: string): HastNode[] {
  // Each word keeps the space after it, so wrapping never changes the text.
  return text.split(/(?<=\s)(?=\S)/u).map((part) =>
    /\S/u.test(part)
      ? {
          type: "element",
          tagName: "span",
          properties: { className: [className] },
          children: [{ type: "text", value: part }],
        }
      : { type: "text", value: part },
  );
}

function wrap(node: HastNode, className: string) {
  if (!node.children) return;
  const next: HastNode[] = [];
  for (const child of node.children) {
    if (child.type === "text" && typeof child.value === "string") {
      next.push(...wordsOf(child.value, className));
    } else {
      if (child.type === "element" && !SKIP.has(child.tagName ?? "") && !isMath(child)) {
        wrap(child, className);
      }
      next.push(child);
    }
  }
  node.children = next;
}

/** The element the reply's last words sit in, where a caret goes after them. */
function lastTextHolder(node: HastNode): HastNode | null {
  const children = node.children ?? [];
  for (let index = children.length - 1; index >= 0; index -= 1) {
    const child = children[index];
    if (child.type === "element" && !SKIP.has(child.tagName ?? "") && !isMath(child)) {
      return lastTextHolder(child) ?? (child.children?.length ? child : null);
    }
    if (child.type === "text" && child.value?.trim()) {
      return node;
    }
  }
  return null;
}

export function rehypeStreamWords(options: { className: string; caret?: string }) {
  return (tree: HastNode) => {
    if (options.className) {
      wrap(tree, options.className);
    }
    if (options.caret) {
      const holder = lastTextHolder(tree);
      holder?.children?.push({
        type: "element",
        tagName: "span",
        properties: { className: [options.caret], ariaHidden: "true" },
        children: [],
      });
    }
  };
}
