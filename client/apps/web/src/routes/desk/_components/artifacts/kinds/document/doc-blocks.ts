/**
 * A document as the Desk reads and edits it: its markdown split into blocks,
 * each kept as the exact text it was written as, so a block can be rewritten
 * or edited and the document put back together without touching the rest.
 */

export type DocBlockType = "sum" | "h" | "p" | "ol" | "ul" | "table" | "quote" | "code" | "rule";

export type DocBlock = {
  /** The block's place in the document, which names it while the text stands. */
  id: string;
  type: DocBlockType;
  /** The block exactly as the markdown wrote it. */
  raw: string;
  /** A heading's level, from 1. */
  level?: number;
  /** A heading's or a paragraph's text, without its markdown prefix. */
  text?: string;
  /** A list's items. */
  items?: string[];
  /** A table's header and rows. */
  header?: string[];
  rows?: string[][];
};

const HEADING = /^(#{1,6})\s+(.*?)\s*#*\s*$/u;
const ORDERED = /^\s*\d+[.)]\s+(.*)$/u;
const BULLET = /^\s*[-*+]\s+(.*)$/u;
const TABLE_RULE = /^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$/u;
const RULE = /^\s*([-*_])(\s*\1){2,}\s*$/u;

function cells(line: string): string[] {
  return line
    .trim()
    .replace(/^\|/u, "")
    .replace(/\|$/u, "")
    .split("|")
    .map((cell) => cell.trim());
}

/** Splits markdown into blocks. The first paragraph before any heading is the summary. */
export function parseDocBlocks(markdown: string): DocBlock[] {
  const lines = markdown.replace(/\r\n/gu, "\n").split("\n");
  const blocks: DocBlock[] = [];
  let index = 0;
  const push = (block: Omit<DocBlock, "id">) => {
    blocks.push({ ...block, id: `b${blocks.length}` });
  };

  while (index < lines.length) {
    const line = lines[index];
    if (line.trim() === "") {
      index++;
      continue;
    }
    const heading = HEADING.exec(line.trim());
    if (heading) {
      push({ type: "h", raw: line, level: heading[1].length, text: heading[2] });
      index++;
      continue;
    }
    if (line.trim().startsWith("```")) {
      const start = index;
      index++;
      while (index < lines.length && !lines[index].trim().startsWith("```")) index++;
      index = Math.min(index + 1, lines.length);
      push({ type: "code", raw: lines.slice(start, index).join("\n") });
      continue;
    }
    if (RULE.test(line)) {
      push({ type: "rule", raw: line });
      index++;
      continue;
    }
    if (
      line.trim().startsWith("|") &&
      index + 1 < lines.length &&
      TABLE_RULE.test(lines[index + 1])
    ) {
      const start = index;
      index += 2;
      while (index < lines.length && lines[index].trim().startsWith("|")) index++;
      const body = lines.slice(start, index);
      push({
        type: "table",
        raw: body.join("\n"),
        header: cells(body[0]),
        rows: body.slice(2).map(cells),
      });
      continue;
    }
    const ordered = ORDERED.test(line);
    if (ordered || BULLET.test(line)) {
      const pattern = ordered ? ORDERED : BULLET;
      const start = index;
      const items: string[] = [];
      while (index < lines.length) {
        const item = pattern.exec(lines[index]);
        if (item) {
          items.push(item[1]);
        } else if (lines[index].startsWith("  ") && lines[index].trim() !== "" && items.length) {
          items[items.length - 1] += ` ${lines[index].trim()}`;
        } else {
          break;
        }
        index++;
      }
      push({ type: ordered ? "ol" : "ul", raw: lines.slice(start, index).join("\n"), items });
      continue;
    }
    if (line.trim().startsWith(">")) {
      const start = index;
      while (index < lines.length && lines[index].trim().startsWith(">")) index++;
      const raw = lines.slice(start, index).join("\n");
      push({
        type: "quote",
        raw,
        text: lines
          .slice(start, index)
          .map((quoted) => quoted.trim().replace(/^>\s?/u, ""))
          .join(" "),
      });
      continue;
    }
    const start = index;
    while (
      index < lines.length &&
      lines[index].trim() !== "" &&
      !HEADING.test(lines[index].trim()) &&
      !lines[index].trim().startsWith("|") &&
      !lines[index].trim().startsWith(">") &&
      !lines[index].trim().startsWith("```") &&
      !ORDERED.test(lines[index]) &&
      !BULLET.test(lines[index])
    ) {
      index++;
    }
    const raw = lines.slice(start, index).join("\n");
    const summary = !blocks.some((block) => block.type !== "rule");
    push({
      type: summary ? "sum" : "p",
      raw,
      text: raw
        .split("\n")
        .map((part) => part.trim())
        .join(" "),
    });
  }

  return blocks;
}

/** Puts blocks back together as markdown, a blank line between each. */
export function joinDocBlocks(blocks: readonly Pick<DocBlock, "raw">[]): string {
  return blocks
    .map((block) => block.raw.trim())
    .filter(Boolean)
    .join("\n\n");
}

/** The document with one block's markdown replaced. */
export function replaceDocBlock(body: string, blockId: string, raw: string): string {
  return joinDocBlocks(
    parseDocBlocks(body).map((block) => (block.id === blockId ? { ...block, raw } : block)),
  );
}

/** Markdown read as plain text: marks out, citations as [N]. */
export function plainText(markdown: string): string {
  return markdown
    .replace(/\[\^(\d+)\]/gu, "[$1]")
    .replace(/\*\*([^*]+)\*\*/gu, "$1")
    .replace(/__([^_]+)__/gu, "$1")
    .replace(/\*([^*]+)\*/gu, "$1")
    .replace(/`([^`]+)`/gu, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/gu, "$1")
    .replace(/^#{1,6}\s+/gmu, "")
    .replace(/^\s*>\s?/gmu, "")
    .replace(/^\s*[-*+]\s+/gmu, "• ");
}

/** How many words a document holds, a table's cells counted as they read. */
export function wordCount(markdown: string): number {
  return plainText(markdown)
    .replace(/\|/gu, " ")
    .replace(/[-:]{3,}/gu, " ")
    .split(/\s+/u)
    .filter((word) => /[\p{L}\p{N}]/u.test(word)).length;
}

/** Minutes to read, at 220 words a minute, never under one. */
export function readMinutes(words: number): number {
  return Math.max(1, Math.round(words / 220));
}

export type InlinePart =
  | { kind: "text"; text: string }
  | { kind: "bold" | "italic" | "code"; text: string }
  | { kind: "link"; text: string; href: string }
  | { kind: "cite"; n: number };

const INLINE =
  /\[\^(\d{1,3})\]|\*\*([^*]+)\*\*|__([^_]+)__|`([^`]+)`|\[([^\]]+)\]\(([^)\s]+)\)|\*([^*\s][^*]*)\*|_([^_\s][^_]*)_/gu;

/** Splits a block's text into its runs: plain, emphasis, code, links and citation marks. */
export function inlineParts(text: string): InlinePart[] {
  const parts: InlinePart[] = [];
  let last = 0;
  for (const match of text.matchAll(INLINE)) {
    const at = match.index ?? 0;
    if (at > last) parts.push({ kind: "text", text: text.slice(last, at) });
    if (match[1]) parts.push({ kind: "cite", n: Number(match[1]) });
    else if (match[2] || match[3]) parts.push({ kind: "bold", text: match[2] ?? match[3] });
    else if (match[4]) parts.push({ kind: "code", text: match[4] });
    else if (match[5]) parts.push({ kind: "link", text: match[5], href: match[6] });
    else parts.push({ kind: "italic", text: match[7] ?? match[8] });
    last = at + match[0].length;
  }
  if (last < text.length) parts.push({ kind: "text", text: text.slice(last) });
  return parts;
}

/**
 * An edited block read back into markdown: bold, italics, code and links
 * written as markdown again, and a citation chip as its mark. Anything else
 * the editor left behind is text.
 */
export function nodeMarkdown(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) {
    return (node.textContent ?? "").replace(/ /gu, " ");
  }
  if (!(node instanceof HTMLElement)) {
    return "";
  }
  const cite = node.dataset.cite;
  if (cite) {
    return `[^${cite}]`;
  }
  const inner = Array.from(node.childNodes).map(nodeMarkdown).join("");
  switch (node.tagName) {
    case "B":
    case "STRONG":
      return inner.trim() ? `**${inner}**` : inner;
    case "I":
    case "EM":
      return inner.trim() ? `*${inner}*` : inner;
    case "CODE":
      return `\`${inner}\``;
    case "A":
      return `[${inner}](${node.getAttribute("href") ?? ""})`;
    case "BR":
      return " ";
    case "DIV":
    case "P":
      return ` ${inner}`;
    default:
      return inner;
  }
}

/** A block as it reads after editing, in the markdown its type is written in. */
export function editedBlockMarkdown(block: DocBlock, element: HTMLElement): string {
  const text = (target: Node) => nodeMarkdown(target).replace(/\s+/gu, " ").trim();
  switch (block.type) {
    case "h":
      return `${"#".repeat(block.level ?? 2)} ${text(element)}`;
    case "ol":
    case "ul": {
      const items = Array.from(element.querySelectorAll("li")).map(text).filter(Boolean);
      return items
        .map((item, at) => (block.type === "ol" ? `${at + 1}. ${item}` : `- ${item}`))
        .join("\n");
    }
    case "quote":
      return `> ${text(element)}`;
    case "sum":
    case "p":
      return text(element);
    default:
      return block.raw;
  }
}
