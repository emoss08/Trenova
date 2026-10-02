import { summaryCount, type ToolStep } from "@/components/assistant/activity";
import { parseToolResult } from "@/components/assistant/tool-presentation";

/** A reply's claim tied to the step that found it: the step's number and where it is cited. */
export type Citation = {
  /** The step's place among the reply's steps, from 1; what the reader sees. */
  n: number;
  step: ToolStep;
  /** Where in the reply the number goes: just after the phrase it backs. */
  offset: number;
};

/** Steps that look nothing up, so there is nothing in the reply for them to back. */
const UNCITED = new Set([
  "ask_user",
  "find_tools",
  "recall_memory",
  "remember",
  "publish_artifact",
  "open_page",
  "raise_exception",
]);

const MAX_LEAVES = 400;
const MIN_TEXT = 3;
const MAX_TEXT = 60;
const ID_LIKE = /^[a-z]{2,}_[0-9a-z]{20,}$/iu;
const ISO_DATE = /^\d{4}-\d{2}-\d{2}/u;

function addLeaf(value: unknown, into: Set<string>) {
  if (typeof value === "string") {
    const text = value.trim();
    if (
      text.length >= MIN_TEXT &&
      text.length <= MAX_TEXT &&
      !ID_LIKE.test(text) &&
      !ISO_DATE.test(text) &&
      /[a-z0-9]/iu.test(text)
    ) {
      into.add(text);
    }
  } else if (typeof value === "number" && Number.isInteger(value) && Math.abs(value) >= 2) {
    into.add(String(value));
  }
}

function walk(value: unknown, into: Set<string>, budget: { left: number }) {
  if (budget.left <= 0) {
    return;
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      walk(item, into, budget);
    }
    return;
  }
  if (value !== null && typeof value === "object") {
    for (const item of Object.values(value)) {
      walk(item, into, budget);
    }
    return;
  }
  budget.left -= 1;
  addLeaf(value, into);
}

/** What a step found that a reply could repeat: the names, numbers and identifiers in its result. */
export function stepAnchors(step: ToolStep): string[] {
  const anchors = new Set<string>();
  const parsed = parseToolResult(step.content);
  if (parsed.kind === "json") {
    walk(parsed.value, anchors, { left: MAX_LEAVES });
  }
  const counted = summaryCount(step.summary);
  if (counted && counted.count >= 2) {
    anchors.add(String(counted.count));
  }

  return [...anchors].sort((a, b) => b.length - a.length);
}

function escape(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

/** The parts of a reply a number must never land in: code spans and links. */
function protectedRanges(text: string): [number, number][] {
  const ranges: [number, number][] = [];
  for (const match of text.matchAll(/`[^`]*`|\[[^\]]*\]\([^)]*\)/gu)) {
    ranges.push([match.index, match.index + match[0].length]);
  }
  return ranges;
}

/** Where a citation for a match ending at `end` goes: past the word, and past the noun a number counts. */
function phraseEnd(text: string, end: number, numeric: boolean): number {
  let at = end;
  while (at < text.length && /[\p{L}\p{N}'-]/u.test(text[at])) {
    at += 1;
  }
  if (numeric) {
    const rest = /^\s+(\p{L}[\p{L}'-]*)/u.exec(text.slice(at));
    if (rest) {
      at += rest[0].length;
    }
  }
  return at;
}

/**
 * Ties each step that found something to the first place the reply repeats
 * it, preferring what only that step found over what several steps share, so
 * two lookups of similar records are told apart. A step whose findings the
 * reply never mentions goes uncited. Matching is on whole words and never
 * inside code or links, so a number lands after the phrase it backs:
 * "15 invoices ①", "Acme ②".
 */
export function citeSteps(text: string, steps: readonly ToolStep[]): Citation[] {
  const blocked = protectedRanges(text);
  const inBlocked = (index: number) => blocked.some(([from, to]) => index >= from && index < to);
  const citable = steps.map(
    (step) =>
      !UNCITED.has(step.name) && step.status !== "failed" && step.content !== "",
  );
  const anchorsByStep = steps.map((step, index) => (citable[index] ? stepAnchors(step) : []));
  const seen = new Map<string, number>();
  for (const anchors of anchorsByStep) {
    for (const anchor of new Set(anchors.map((value) => value.toLowerCase()))) {
      seen.set(anchor, (seen.get(anchor) ?? 0) + 1);
    }
  }

  const firstMention = (anchors: readonly string[]) => {
    let best: { start: number; offset: number } | null = null;
    for (const anchor of anchors) {
      const numeric = /^\d+$/u.test(anchor);
      const pattern = new RegExp(`(?<![\\p{L}\\p{N}])${escape(anchor)}(?![\\p{L}\\p{N}])`, "giu");
      for (const match of text.matchAll(pattern)) {
        if (inBlocked(match.index)) {
          continue;
        }
        if (!best || match.index < best.start) {
          best = {
            start: match.index,
            offset: phraseEnd(text, match.index + match[0].length, numeric),
          };
        }
        break;
      }
    }
    return best;
  };

  const citations: Citation[] = [];
  steps.forEach((step, index) => {
    if (!citable[index]) {
      return;
    }
    const anchors = anchorsByStep[index];
    const own = anchors.filter((anchor) => seen.get(anchor.toLowerCase()) === 1);
    const best = firstMention(own) ?? firstMention(anchors);
    if (best) {
      citations.push({ n: index + 1, step, offset: best.offset });
    }
  });

  return citations.sort((a, b) => a.offset - b.offset || a.n - b.n);
}

/** The prefix a citation link carries, so the reply's renderer can tell it from a real link. */
export const CITATION_HREF = "#dk-cite-";

/** The reply with each citation written in as a link the renderer draws as its number. */
export function withCitations(text: string, citations: readonly Citation[]): string {
  let out = "";
  let from = 0;
  for (const citation of citations) {
    out += text.slice(from, citation.offset);
    out += `[${citation.n}](${CITATION_HREF}${citation.n})`;
    from = citation.offset;
  }

  return out + text.slice(from);
}
