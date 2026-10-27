// x12.ts colours X12 as it is written: each segment's ID (the envelope's ISA, GS, ST, SE, GE and
// IEA apart from the transaction's own), the separators between elements, components and
// repetitions, the segment terminator, and numbers. The separators are the interchange's own,
// read from its ISA segment; text without one uses the usual `*`, `>`, `^` and `~`.

import type { CodeRole, CodeSpan } from "@trenova/shared/lib/highlight/code";

export type X12Separators = {
  element: string;
  component: string;
  repetition: string;
  segment: string;
};

const DEFAULT_SEPARATORS: X12Separators = {
  element: "*",
  component: ">",
  repetition: "^",
  segment: "~",
};

const ENVELOPE = new Set(["ISA", "GS", "ST", "SE", "GE", "IEA"]);

/** An ISA's elements: the component separator is the 16th and the terminator follows it. */
const ISA_ELEMENTS = 16;

/** Past this many characters a text is shown plain: one element per span would swamp the page. */
export const X12_HIGHLIGHT_LIMIT = 200_000;

/**
 * The separators an interchange declares in its ISA segment, or the usual ones without one. The
 * ISA is read by its elements, not its fixed width, so a squeezed ISA reads the same.
 */
export function x12Separators(text: string): X12Separators {
  const start = text.search(/\S/);
  if (start < 0 || !text.startsWith("ISA", start)) return DEFAULT_SEPARATORS;
  const element = text.charAt(start + 3);
  if (element === "" || /[A-Za-z0-9\s]/.test(element)) return DEFAULT_SEPARATORS;
  const fields: string[] = [];
  let from = start + 4;
  while (fields.length < ISA_ELEMENTS - 1) {
    const next = text.indexOf(element, from);
    if (next < 0) return DEFAULT_SEPARATORS;
    fields.push(text.slice(from, next));
    from = next + 1;
  }
  const component = text.charAt(from);
  const segment = text.charAt(from + 1);
  if (component === "" || segment === "" || /[A-Za-z0-9]/.test(segment)) {
    return DEFAULT_SEPARATORS;
  }
  const repetition = fields[10] ?? "";
  return {
    element,
    component,
    repetition:
      repetition.length === 1 && !/[A-Za-z0-9]/.test(repetition)
        ? repetition
        : DEFAULT_SEPARATORS.repetition,
    segment: /\s/.test(segment) ? "\n" : segment,
  };
}

/** Whether a text reads as X12: an ISA, or a first segment ID followed by its separator. */
export function looksLikeX12(text: string): boolean {
  const start = text.search(/\S/);
  if (start < 0) return false;
  if (text.startsWith("ISA", start)) return true;
  return /^[A-Z][A-Z0-9]{1,2}[*|^~]/.test(text.slice(start, start + 4));
}

const NUMBER = /^-?\d+(\.\d+)?$/;

/**
 * The spans `HighlightedCode` and the editors colour X12 with; none when it is not X12, or is
 * longer than `X12_HIGHLIGHT_LIMIT`.
 */
export function x12Spans(text: string): CodeSpan[] {
  if (text.length > X12_HIGHLIGHT_LIMIT || !looksLikeX12(text)) return [];
  const separators = x12Separators(text);
  const breaks = new Set([
    separators.element,
    separators.component,
    separators.repetition,
    separators.segment,
  ]);
  const spans: CodeSpan[] = [];
  const push = (from: number, to: number, role: CodeRole) => {
    if (to > from) spans.push({ from, to, role });
  };
  let at = 0;
  let atSegmentStart = true;
  while (at < text.length) {
    const char = text.charAt(at);
    if (char === separators.segment) {
      if (!/\s/.test(char)) push(at, at + 1, "operator");
      at += 1;
      atSegmentStart = true;
      continue;
    }
    if (/\s/.test(char) && (atSegmentStart || char === "\n" || char === "\r")) {
      at += 1;
      continue;
    }
    if (breaks.has(char)) {
      push(at, at + 1, "operator");
      at += 1;
      continue;
    }
    let end = at;
    while (end < text.length && !breaks.has(text.charAt(end)) && text.charAt(end) !== "\n") {
      end += 1;
    }
    const value = text.slice(at, end);
    if (atSegmentStart) {
      push(at, end, ENVELOPE.has(value) ? "meta" : "keyword");
      atSegmentStart = false;
    } else if (NUMBER.test(value)) {
      push(at, end, "number");
    }
    at = end;
  }
  return spans;
}
