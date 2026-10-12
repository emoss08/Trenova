import type { TriageFact } from "@/lib/graphql/inbox";

/**
 * How the reader reads a message's words: the sender's own text with the
 * thread they replied on folded away, the paragraphs of it with the words
 * Nova read a fact from marked, and whether it is raw EDI rather than prose.
 */

/**
 * A message body split the way a mail client shows it: what this sender
 * wrote, and the thread they replied on top of, folded away until asked for.
 *
 * The fold starts at a reply line ("On … wrote:"), a forwarded or original
 * message header, or a trailing run of ">" lines. A quote the sender answers
 * line by line is their message, so it stays. The boundaries are the same
 * ones the server's list preview stops at.
 */
export type QuotedBody = { own: string; quoted: string };

const REPLY_INTRO = /^On .{4,200} wrote:$/;
const HEADER_PREFIXES = [
  "-----Original Message-----",
  "---------- Forwarded message",
  "________________________________",
];

function isFoldStart(line: string): boolean {
  const trimmed = line.trim();

  return REPLY_INTRO.test(trimmed) || HEADER_PREFIXES.some((prefix) => trimmed.startsWith(prefix));
}

function isQuoteOrBlank(line: string): boolean {
  const trimmed = line.trim();

  return trimmed === "" || trimmed.startsWith(">");
}

function trimBlankEdges(lines: string[]): string {
  return lines
    .join("\n")
    .replace(/^\s*\n/, "")
    .trimEnd();
}

export function splitQuotedBody(body: string): QuotedBody {
  const lines = body.replace(/\r\n?/g, "\n").split("\n");

  let fold = lines.findIndex(isFoldStart);
  if (fold === -1) {
    fold = lines.length;
    while (fold > 0 && isQuoteOrBlank(lines[fold - 1])) {
      fold--;
    }
    if (!lines.slice(fold).some((line) => line.trim().startsWith(">"))) {
      fold = lines.length;
    }
  }

  const own = trimBlankEdges(lines.slice(0, fold));
  const quoted = trimBlankEdges(lines.slice(fold));
  if (own === "") {
    return { own: trimBlankEdges(lines), quoted: "" };
  }

  return { own, quoted };
}

/** A fact as the body marks it: the words it was read from, and what it was read as. */
export type UnderlineFact = Pick<TriageFact, "key" | "label" | "quote" | "tone" | "flagged">;

export type BodySegment =
  | { kind: "text"; text: string }
  | { kind: "fact"; text: string; fact: UnderlineFact };

type Mark = { start: number; end: number; fact: UnderlineFact };

const PARAGRAPH_BREAK = /\n[ \t]*\n\s*/g;
const REGEXP_SPECIAL = /[.*+?^${}()|[\]\\]/g;

/**
 * The quote as a pattern that finds it however the message wraps it: any run
 * of whitespace in the quote matches any run in the words, and case is
 * ignored, the way the server checked that the quote is there at all.
 */
function quotePattern(quote: string): RegExp | null {
  const words = quote.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return null;
  }

  return new RegExp(words.map((word) => word.replace(REGEXP_SPECIAL, "\\$&")).join("\\s+"), "i");
}

/**
 * Where each fact sits in the words: the first place its quote appears that
 * no fact before it has taken. A fact whose quote is missing is left out.
 */
function factMarks(text: string, facts: readonly UnderlineFact[]): Mark[] {
  const marks: Mark[] = [];
  for (const fact of facts) {
    const pattern = quotePattern(fact.quote);
    if (pattern === null) {
      continue;
    }
    let from = 0;
    while (from <= text.length) {
      const found = pattern.exec(text.slice(from));
      if (found === null) {
        break;
      }
      const start = from + found.index;
      const end = start + found[0].length;
      const clashes = marks.some((mark) => start < mark.end && mark.start < end);
      if (!clashes) {
        marks.push({ start, end, fact });
        break;
      }
      from = start + 1;
    }
  }

  return marks.sort((a, b) => a.start - b.start);
}

function segmentsOf(text: string, offset: number, marks: readonly Mark[]): BodySegment[] {
  const segments: BodySegment[] = [];
  let cursor = 0;
  for (const mark of marks) {
    const start = mark.start - offset;
    const end = mark.end - offset;
    if (start > cursor) {
      segments.push({ kind: "text", text: text.slice(cursor, start) });
    }
    segments.push({ kind: "fact", text: text.slice(start, end), fact: mark.fact });
    cursor = end;
  }
  if (cursor < text.length) {
    segments.push({ kind: "text", text: text.slice(cursor) });
  }

  return segments;
}

/**
 * The words as paragraphs, split at blank lines, with every fact Nova read
 * marked where its quote is. A quote that runs across a paragraph break is
 * not marked, because no one underline can cover it.
 */
export function underlineParagraphs(
  text: string,
  facts: readonly UnderlineFact[],
): BodySegment[][] {
  const marks = factMarks(text, facts);
  const paragraphs: BodySegment[][] = [];
  let start = 0;
  const pushParagraph = (end: number) => {
    const raw = text.slice(start, end);
    const lead = raw.length - raw.trimStart().length;
    const body = raw.trim();
    if (body !== "") {
      const from = start + lead;
      const to = from + body.length;
      const inside = marks.filter((mark) => mark.start >= from && mark.end <= to);
      paragraphs.push(segmentsOf(body, from, inside));
    }
  };

  for (const found of text.matchAll(PARAGRAPH_BREAK)) {
    pushParagraph(found.index);
    start = found.index + found[0].length;
  }
  pushParagraph(text.length);

  return paragraphs;
}

const EDI_SEGMENT = /\b[A-Z][A-Z0-9]{1,2}\*[^~\n]*~/g;

/** Whether the words are an X12 document (segments ending in "~") rather than prose. */
export function isRawEdi(text: string): boolean {
  return (text.match(EDI_SEGMENT)?.length ?? 0) >= 2;
}
