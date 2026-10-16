/**
 * Matches what a person types against what something is called, the way a
 * launcher should: every typed word has to land at the start of the field,
 * at the start of a word inside it, or on its words' initials. Only a word of
 * three or more characters may land inside a word, so "ai" finds "AI control"
 * and "Aisha Grant" but never "Trailers" or "Email logs".
 *
 * A match says which field each word won and exactly which characters it
 * used, so a row can mark those characters and nothing else.
 */
export type MatchRange = readonly [start: number, end: number];

export interface MatchField<K extends string = string> {
  key: K;
  text: string;
  /** Scales everything the field matches; a name outranks a breadcrumb. */
  weight?: number;
  /**
   * Set on a field the row does not draw, such as a page keyword or a
   * shipment's driver. A match through it is shown as its own tag,
   * "Keyword assistant", so the row says why it is there.
   */
  via?: string;
}

export interface TermMatch {
  score: number;
  ranges: MatchRange[];
}

export interface MatchVia {
  label: string;
  text: string;
  ranges: MatchRange[];
}

export interface FieldsMatch<K extends string = string> {
  score: number;
  /** The merged characters each drawn field won. A field no word won has no entry. */
  highlights: Partial<Record<K, MatchRange[]>>;
  /** The first hidden field a word won, with what it matched. */
  via: MatchVia | null;
}

export const FIELD_PREFIX_SCORE = 1;
export const WORD_PREFIX_SCORE = 0.9;
export const INITIALS_SCORE = 0.72;
export const INSIDE_WORD_SCORE = 0.5;
export const TITLE_PHRASE_SCORE = 1.05;
export const ALL_ON_TITLE_BONUS = 0.04;
export const MIN_INSIDE_WORD_LENGTH = 3;

const WORD_OFFSET_PENALTY = 0.002;
const MAX_WORD_OFFSET = 40;
const LETTER = /\p{L}/u;
const DIGIT = /\p{N}/u;
const LOWER = /\p{Ll}/u;
const UPPER = /\p{Lu}/u;

function isAlphanumeric(char: string): boolean {
  return LETTER.test(char) || DIGIT.test(char);
}

/**
 * Folds one character for comparison: lower case without its accent. A
 * character that would fold to more than one (a ligature) keeps its own lower
 * case, so every index in the folded text is an index in the original.
 */
function foldChar(char: string): string {
  const folded = char.normalize("NFKD").replace(/\p{M}/gu, "").toLowerCase();
  if (folded.length === 1) {
    return folded;
  }
  const lower = char.toLowerCase();
  return lower.length === 1 ? lower : char;
}

function foldText(text: string): string {
  let folded = "";
  for (let i = 0; i < text.length; i++) {
    folded += foldChar(text.charAt(i));
  }
  return folded;
}

export function foldQuery(query: string): string {
  return query.normalize("NFKD").replace(/\p{M}/gu, "").toLowerCase().trim().replace(/\s+/g, " ");
}

/**
 * Whether a word starts at this index: the first character, anything after a
 * character that is not a letter or digit, a lower-to-upper case change
 * ("billingQueue"), and a letter-to-digit change, so "240193" lands on the
 * number in "S-240193" and in "S240193".
 */
export function isWordStart(text: string, index: number): boolean {
  if (index === 0) {
    return true;
  }
  const previous = text.charAt(index - 1);
  const current = text.charAt(index);
  if (!isAlphanumeric(previous)) {
    return true;
  }
  if (LOWER.test(previous) && UPPER.test(current)) {
    return true;
  }
  return LETTER.test(previous) && DIGIT.test(current);
}

function matchFolded(term: string, text: string, folded: string): TermMatch | null {
  if (term === "" || text === "") {
    return null;
  }

  if (folded.startsWith(term)) {
    return { score: FIELD_PREFIX_SCORE, ranges: [[0, term.length]] };
  }

  for (let i = 1; i <= folded.length - term.length; i++) {
    if (isWordStart(text, i) && folded.startsWith(term, i)) {
      return {
        score: WORD_PREFIX_SCORE - Math.min(i, MAX_WORD_OFFSET) * WORD_OFFSET_PENALTY,
        ranges: [[i, i + term.length]],
      };
    }
  }

  if (term.length >= 2) {
    let starts = 0;
    let cursor = 0;
    const ranges: MatchRange[] = [];
    for (let i = 0; i < text.length; i++) {
      if (!isWordStart(text, i) || !isAlphanumeric(text.charAt(i))) {
        continue;
      }
      starts++;
      if (cursor < term.length && folded.charAt(i) === term.charAt(cursor)) {
        ranges.push([i, i + 1]);
        cursor++;
      }
    }
    if (cursor === term.length && starts > 1) {
      return { score: INITIALS_SCORE, ranges };
    }
  }

  if (term.length >= MIN_INSIDE_WORD_LENGTH) {
    const index = folded.indexOf(term);
    if (index >= 0) {
      return { score: INSIDE_WORD_SCORE, ranges: [[index, index + term.length]] };
    }
  }

  return null;
}

/**
 * One typed word against one field, best rule first: the field's start, a
 * later word's start (a little less the further in), the words' initials,
 * then, for three or more characters only, anywhere inside a word.
 */
export function matchTerm(term: string, text: string): TermMatch | null {
  return matchFolded(foldQuery(term), text, foldText(text));
}

/** Sorted, with overlapping and touching ranges joined into one. */
export function mergeRanges(ranges: readonly MatchRange[]): MatchRange[] {
  if (ranges.length === 0) {
    return [];
  }
  const sorted = [...ranges].sort((a, b) => a[0] - b[0] || a[1] - b[1]);
  const merged: [number, number][] = [];
  for (const [start, end] of sorted) {
    const last = merged[merged.length - 1];
    if (last && start <= last[1]) {
      last[1] = Math.max(last[1], end);
    } else {
      merged.push([start, end]);
    }
  }
  return merged;
}

const EMPTY_MATCH: FieldsMatch<never> = { score: 1, highlights: {}, via: null };

/**
 * A query against one thing's fields, the first field being its title. Every
 * word has to land somewhere or nothing matches. The score is the mean of
 * each word's best match times its field's weight, a little more when every
 * word landed on the title, and highest of all when a phrase of several
 * words is how the title begins.
 */
export function matchFields<K extends string>(
  query: string,
  fields: readonly MatchField<K>[],
): FieldsMatch<K> | null {
  const folded = foldQuery(query);
  if (folded === "") {
    return EMPTY_MATCH;
  }

  const prepared = fields
    .filter((field) => field.text.length > 0)
    .map((field) => ({ field, folded: foldText(field.text), weight: field.weight ?? 1 }));
  if (prepared.length === 0) {
    return null;
  }

  const title = fields[0];
  const titleEntry = prepared[0]?.field === title ? prepared[0] : undefined;
  if (titleEntry && folded.includes(" ") && titleEntry.folded.startsWith(folded)) {
    const highlights: Partial<Record<K, MatchRange[]>> = {};
    highlights[titleEntry.field.key] = [[0, folded.length]];
    return { score: TITLE_PHRASE_SCORE, highlights, via: null };
  }

  const terms = folded.split(" ");
  const won = new Map<K, MatchRange[]>();
  let via: MatchVia | null = null;
  let total = 0;
  let onTitle = 0;

  for (const term of terms) {
    let best: { value: number; entry: (typeof prepared)[number]; ranges: MatchRange[] } | null =
      null;
    for (const entry of prepared) {
      const match = matchFolded(term, entry.field.text, entry.folded);
      if (!match) {
        continue;
      }
      const value = match.score * entry.weight;
      if (!best || value > best.value) {
        best = { value, entry, ranges: match.ranges };
      }
    }
    if (!best) {
      return null;
    }

    total += best.value;
    if (best.entry === titleEntry) {
      onTitle++;
    }
    const { field } = best.entry;
    if (field.via) {
      via ??= { label: field.via, text: field.text, ranges: best.ranges };
    } else {
      won.set(field.key, [...(won.get(field.key) ?? []), ...best.ranges]);
    }
  }

  const highlights: Partial<Record<K, MatchRange[]>> = {};
  for (const [key, ranges] of won) {
    highlights[key] = mergeRanges(ranges);
  }

  return {
    score: total / terms.length + (onTitle === terms.length ? ALL_ON_TITLE_BONUS : 0),
    highlights,
    via,
  };
}

function sameRanges(a: readonly MatchRange[], b: readonly MatchRange[]): boolean {
  return (
    a === b ||
    (a.length === b.length &&
      a.every((range, i) => range[0] === b[i]?.[0] && range[1] === b[i]?.[1]))
  );
}

/**
 * Whether two matches mark the same characters through the same field. The
 * score is left out: a row draws what a match marks, not how well it ranked,
 * so a list re-ranked for a longer query redraws only the rows whose marks moved.
 */
export function sameFieldsMatch<K extends string>(
  a: FieldsMatch<K> | null | undefined,
  b: FieldsMatch<K> | null | undefined,
): boolean {
  if (a === b) {
    return true;
  }
  if (!a || !b) {
    return false;
  }
  const keys = Object.keys(a.highlights) as K[];
  if (keys.length !== Object.keys(b.highlights).length) {
    return false;
  }
  for (const key of keys) {
    const other = b.highlights[key];
    if (!other || !sameRanges(a.highlights[key] ?? [], other)) {
      return false;
    }
  }
  if (a.via === b.via) {
    return true;
  }
  return (
    a.via !== null &&
    b.via !== null &&
    a.via.label === b.via.label &&
    a.via.text === b.via.text &&
    sameRanges(a.via.ranges, b.via.ranges)
  );
}

export interface RankedMatch<T, K extends string = string> {
  entry: T;
  match: FieldsMatch<K>;
}

/**
 * The entries that match, best first. Ties keep their original order, so a
 * list that was arranged on purpose stays arranged, and an empty query keeps
 * everything as it was.
 */
export function rankMatches<T, K extends string>(
  query: string,
  entries: readonly T[],
  fieldsOf: (entry: T) => readonly MatchField<K>[],
): RankedMatch<T, K>[] {
  if (foldQuery(query) === "") {
    return entries.map((entry) => ({ entry, match: EMPTY_MATCH }));
  }

  const ranked: (RankedMatch<T, K> & { index: number })[] = [];
  entries.forEach((entry, index) => {
    const match = matchFields(query, fieldsOf(entry));
    if (match) {
      ranked.push({ entry, match, index });
    }
  });
  ranked.sort((a, b) => b.match.score - a.match.score || a.index - b.index);
  return ranked.map(({ entry, match }) => ({ entry, match }));
}
