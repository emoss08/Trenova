import type { AssistantEntityRef } from "@/types/assistant";
import { escapeRegExp } from "@trenova/shared/lib/regexp";
import type { JSONContent } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";

/**
 * The composer's document is plain text: one paragraph a line, nothing else.
 * What the rest of the Desk reads and writes is that text, lines joined by a
 * newline, and a caret is an offset into it. These map between the two, so
 * the menus that read "the draft and where the caret is" never see a
 * document position.
 */

/** The document for a draft: one paragraph for each of its lines. */
export function docOf(text: string): JSONContent {
  return {
    type: "doc",
    content: text.split("\n").map((line) => ({
      type: "paragraph",
      content: line === "" ? [] : [{ type: "text", text: line }],
    })),
  };
}

/** The draft a document holds: its paragraphs' text, joined by newlines. */
export function textOf(doc: ProseMirrorNode): string {
  const lines: string[] = [];
  doc.forEach((block) => {
    lines.push(block.textContent);
  });
  return lines.join("\n");
}

/** The document position of an offset into the draft, held inside the document. */
export function posAt(doc: ProseMirrorNode, offset: number): number {
  let left = Math.max(0, offset);
  let found: number | null = null;
  let end = 1;
  doc.forEach((block, start) => {
    if (found !== null) {
      return;
    }
    const size = block.content.size;
    if (left <= size) {
      found = start + 1 + left;
      return;
    }
    left -= size + 1;
    end = start + 1 + size;
  });
  return found ?? end;
}

/** The offset into the draft of a document position. */
export function offsetAt(doc: ProseMirrorNode, pos: number): number {
  let offset = 0;
  let found: number | null = null;
  doc.forEach((block, start) => {
    if (found !== null) {
      return;
    }
    const size = block.content.size;
    const inside = start + 1;
    if (pos <= inside + size) {
      found = offset + Math.max(0, pos - inside);
      return;
    }
    offset += size + 1;
  });
  return found ?? Math.max(0, offset - 1);
}

/** A stretch of the draft, as offsets into it. */
export type TextRange = { from: number; to: number };

/**
 * Where each named record's @label stands in the draft, longest label first
 * so "@Acme Freight West" is not read as "@Acme Freight". A label that is
 * the start of a longer word is not a mention of it.
 */
export function mentionRanges(text: string, mentions: readonly AssistantEntityRef[]): TextRange[] {
  const labels = [
    ...new Set(mentions.map((mention) => mention.label).filter((label) => label !== "")),
  ];
  if (labels.length === 0 || !text.includes("@")) {
    return [];
  }
  const pattern = new RegExp(
    "@(?:" +
      labels
        .sort((a, b) => b.length - a.length)
        .map(escapeRegExp)
        .join("|") +
      ")",
    "g",
  );
  const ranges: TextRange[] = [];
  for (const match of text.matchAll(pattern)) {
    const from = match.index ?? 0;
    ranges.push({ from, to: from + match[0].length });
  }
  return ranges;
}

/** The mention a caret sits at the end of, so one Backspace takes the whole name. */
export function mentionEndingAt(
  text: string,
  mentions: readonly AssistantEntityRef[],
  caret: number,
): TextRange | null {
  return mentionRanges(text, mentions).find((range) => range.to === caret) ?? null;
}

/** Text past this is a document, not a message: it is attached as a file where files are taken. */
export const PASTE_AS_FILE_CHARS = 6000;
/** A paste of more lines than this is attached as a file where files are taken. */
export const PASTE_AS_FILE_LINES = 80;

/** Whether pasted text is long enough to ride as a file rather than fill the box. */
export function pastesAsFile(text: string): boolean {
  if (text.length > PASTE_AS_FILE_CHARS) {
    return true;
  }
  let lines = 1;
  for (let index = text.indexOf("\n"); index !== -1; index = text.indexOf("\n", index + 1)) {
    lines += 1;
    if (lines > PASTE_AS_FILE_LINES) {
      return true;
    }
  }
  return false;
}

const INVISIBLE = /[​-‍⁠﻿]/g;

/**
 * Pasted text as a message wants it: one kind of line ending, no characters
 * that cannot be seen, non-breaking spaces as spaces, a spreadsheet's cells
 * two spaces apart instead of tab-jumped, no trailing spaces, at most one
 * blank line in a row and none at either end.
 */
export function cleanPastedText(raw: string): string {
  return raw
    .replace(/\r\n?/g, "\n")
    .replace(INVISIBLE, "")
    .replace(/ /g, " ")
    .replace(/\t/g, "  ")
    .replace(/[ ]+$/gm, "")
    .replace(/\n{3,}/g, "\n\n")
    .replace(/^\n+|\n+$/g, "");
}

const BLOCK_TAGS = new Set([
  "ADDRESS",
  "ARTICLE",
  "ASIDE",
  "BLOCKQUOTE",
  "DIV",
  "DL",
  "DT",
  "DD",
  "FOOTER",
  "H1",
  "H2",
  "H3",
  "H4",
  "H5",
  "H6",
  "HEADER",
  "LI",
  "OL",
  "P",
  "PRE",
  "SECTION",
  "TABLE",
  "TR",
  "UL",
]);

/**
 * The text of pasted HTML for a clipboard that carries no plain text of its
 * own: each block on a line of its own, list items marked, a table's cells
 * set apart, nothing from a script or a style sheet.
 */
export function textOfHtml(html: string): string {
  const parsed = new DOMParser().parseFromString(html, "text/html");
  const out: string[] = [];
  const walk = (node: Node, list: { ordered: boolean; index: number } | null) => {
    if (node.nodeType === Node.TEXT_NODE) {
      out.push((node.textContent ?? "").replace(/\s+/g, " "));
      return;
    }
    if (!(node instanceof HTMLElement)) {
      return;
    }
    const tag = node.tagName;
    if (tag === "SCRIPT" || tag === "STYLE" || tag === "HEAD" || tag === "TEMPLATE") {
      return;
    }
    if (tag === "BR") {
      out.push("\n");
      return;
    }
    const block = BLOCK_TAGS.has(tag);
    if (block) {
      out.push("\n");
    }
    if (tag === "LI") {
      if (list?.ordered) {
        list.index += 1;
        out.push(`${list.index}. `);
      } else {
        out.push("- ");
      }
    }
    const inner =
      tag === "OL"
        ? { ordered: true, index: 0 }
        : tag === "UL"
          ? { ordered: false, index: 0 }
          : list;
    for (const child of node.childNodes) {
      walk(child, inner);
    }
    if (tag === "TD" || tag === "TH") {
      out.push("\t");
    }
    if (block) {
      out.push("\n");
    }
  };
  walk(parsed.body, null);
  return out
    .join("")
    .replace(/[ \t]*\n[ \t]*/g, "\n")
    .replace(/\t+$/gm, "")
    .replace(/\n{2,}/g, "\n");
}
