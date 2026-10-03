/**
 * Keeps a reply's markdown to the set the Desk draws. GFM brings a few things
 * the design leaves out on purpose: a bare address turning into a link,
 * footnotes, images, and code set off by indentation rather than a fence.
 * Each is put back as the words that were written, so the reply reads as
 * typed rather than as something the reader did not expect.
 *
 * What a node was written as is read from the source: a link the author
 * wrote starts with "[" or "<", a fenced block with ``` or ~~~.
 */

type Point = { offset?: number };
type MdNode = {
  type: string;
  value?: string;
  url?: string;
  children?: MdNode[];
  position?: { start: Point; end: Point };
};

function sourceOf(node: MdNode, source: string): string | null {
  const start = node.position?.start.offset;
  const end = node.position?.end.offset;
  if (start === undefined || end === undefined) return null;
  return source.slice(start, end);
}

function literal(node: MdNode, source: string): MdNode {
  return { type: "text", value: sourceOf(node, source) ?? node.value ?? node.url ?? "" };
}

function rewrite(node: MdNode, source: string): MdNode[] {
  const written = sourceOf(node, source);
  switch (node.type) {
    case "link":
      // A bare address GFM made into a link: written without brackets.
      if (written !== null && !written.startsWith("[") && !written.startsWith("<")) {
        return [{ type: "text", value: written }];
      }
      break;
    case "image":
    case "imageReference":
    case "footnoteReference":
      return [literal(node, source)];
    case "footnoteDefinition":
      return [{ type: "paragraph", children: [literal(node, source)] }];
    case "code":
      // Indented code: shown as the paragraph it was typed as.
      if (written !== null && !/^\s{0,3}(`{3,}|~{3,})/u.test(written)) {
        return [{ type: "paragraph", children: [{ type: "text", value: written.trim() }] }];
      }
      break;
    default:
      break;
  }
  if (node.children) {
    node.children = node.children.flatMap((child) => rewrite(child, source));
  }
  return [node];
}

export function remarkDeskSubset() {
  return (tree: MdNode, file: { value?: unknown }) => {
    const source = typeof file.value === "string" ? file.value : String(file.value ?? "");
    tree.children = (tree.children ?? []).flatMap((child) => rewrite(child, source));
  };
}
