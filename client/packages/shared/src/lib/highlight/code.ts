import {
  createHighlighter,
  defineLanguage,
  type HighlightTokenClass,
} from "@tanstack/highlight/core";
import { diff } from "@tanstack/highlight/languages/diff";
import { js } from "@tanstack/highlight/languages/js";
import { json } from "@tanstack/highlight/languages/json";
import { python } from "@tanstack/highlight/languages/python";
import { shell } from "@tanstack/highlight/languages/shell";
import { sql } from "@tanstack/highlight/languages/sql";
import { ts } from "@tanstack/highlight/languages/ts";
import { yaml } from "@tanstack/highlight/languages/yaml";
import { escapeRegExp } from "@trenova/shared/lib/regexp";
import { graphql } from "./graphql";

// The languages the app shows code in: JSON payloads and audit records, and whatever an agent
// writes in a fenced block. Each grammar is a few hundred bytes, and highlighting is
// synchronous, so a block is coloured on its first paint rather than after a fetch.
const highlighter = createHighlighter({
  languages: [
    json,
    js,
    ts,
    defineLanguage({
      ...sql,
      aliases: ["mysql", "pgsql", "plsql", "postgres", "postgresql", "psql", "tsql"],
    }),
    graphql,
    python,
    yaml,
    shell,
    diff,
  ],
});

const PLAIN = "plaintext";

export type CodeRole = HighlightTokenClass;

/** A run of code with one syntactic role, or none for text between tokens. */
export type CodeSegment = { readonly role?: CodeRole; readonly text: string };

export type CodeLine = readonly CodeSegment[];

/**
 * codeLanguage names the grammar a label ("sql", "JSON", "gql") is highlighted with, or null
 * when the app has none for it and the code is shown as plain text.
 */
export function codeLanguage(label: string | undefined): string | null {
  if (!label) return null;
  const language = highlighter.normalizeLanguage(label.toLowerCase());
  return language === PLAIN ? null : language;
}

/** highlightLines splits code into lines, each a run of segments coloured by role. */
export function highlightLines(code: string, label?: string): CodeLine[] {
  const language = codeLanguage(label);
  const tokens = language
    ? highlighter.tokenize(code, { lang: language }).tokens
    : [{ value: code, className: undefined }];

  const lines: CodeSegment[][] = [[]];
  for (const token of tokens) {
    const parts = token.value.split("\n");
    for (let i = 0; i < parts.length; i += 1) {
      if (i > 0) lines.push([]);
      const text = parts[i];
      if (text === "") continue;
      lines[lines.length - 1].push(token.className ? { role: token.className, text } : { text });
    }
  }
  return lines;
}

/** A range of code with a role, in UTF-16 code units, as a language's own tokenizer gives it. */
export type CodeSpan = { readonly from: number; readonly to: number; readonly role?: CodeRole };

/**
 * highlightSpans splits code into lines coloured by spans another tokenizer produced (the rules
 * module's `tariff.tokens`), with the text between spans left plain. Spans must be in order and
 * not overlap; one that runs past the end of the code is cut there.
 */
export function highlightSpans(code: string, spans: readonly CodeSpan[]): CodeLine[] {
  const segments: CodeSegment[] = [];
  let at = 0;
  for (const span of spans) {
    const from = Math.max(span.from, at);
    const to = Math.min(span.to, code.length);
    if (to <= from) continue;
    if (from > at) segments.push({ text: code.slice(at, from) });
    const text = code.slice(from, to);
    segments.push(span.role ? { role: span.role, text } : { text });
    at = to;
  }
  if (at < code.length) segments.push({ text: code.slice(at) });

  const lines: CodeSegment[][] = [[]];
  for (const segment of segments) {
    const parts = segment.text.split("\n");
    for (let i = 0; i < parts.length; i += 1) {
      if (i > 0) lines.push([]);
      const text = parts[i];
      if (text === "") continue;
      lines[lines.length - 1].push(segment.role ? { role: segment.role, text } : { text });
    }
  }
  return lines;
}

export type MatchRange = readonly [start: number, end: number];

/** createMatcher finds a query anywhere, ignoring case; a blank query finds nothing. */
export function createMatcher(query: string): RegExp | null {
  return query.trim() === "" ? null : new RegExp(escapeRegExp(query), "gi");
}

export function findMatches(text: string, matcher: RegExp | null): MatchRange[] {
  if (matcher === null) return [];
  const ranges: MatchRange[] = [];
  for (const match of text.matchAll(matcher)) {
    ranges.push([match.index, match.index + match[0].length]);
  }
  return ranges;
}

/** countMatches counts what findMatches marks line by line; a match never spans lines. */
export function countMatches(code: string, matcher: RegExp | null): number {
  if (matcher === null) return 0;
  let count = 0;
  for (const line of code.split("\n")) count += findMatches(line, matcher).length;
  return count;
}

export function lineText(line: CodeLine): string {
  let text = "";
  for (const segment of line) text += segment.text;
  return text;
}

export type MarkedSegment = CodeSegment & { readonly marked: boolean };

/**
 * markSegments cuts a line's segments at the edges of each match, so a match that starts in a
 * property and ends in the string after it is marked across both.
 */
export function markSegments(line: CodeLine, matches: readonly MatchRange[]): MarkedSegment[] {
  if (matches.length === 0) return line.map((segment) => ({ ...segment, marked: false }));

  const out: MarkedSegment[] = [];
  let offset = 0;
  let next = 0;
  for (const segment of line) {
    const end = offset + segment.text.length;
    let cursor = offset;
    while (cursor < end) {
      while (next < matches.length && matches[next][1] <= cursor) next += 1;
      const match = matches[next];
      const inMatch = match !== undefined && match[0] <= cursor;
      const stop = inMatch ? Math.min(match[1], end) : Math.min(match?.[0] ?? end, end);
      out.push({
        ...(segment.role ? { role: segment.role } : {}),
        text: segment.text.slice(cursor - offset, stop - offset),
        marked: inMatch,
      });
      cursor = stop;
    }
    offset = end;
  }
  return out;
}
