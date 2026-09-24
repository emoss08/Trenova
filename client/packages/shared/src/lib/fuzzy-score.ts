/**
 * Ranks free text against a query the way a launcher should: a match at the
 * start of what something is called beats a match at the start of a word
 * inside it, which beats a match anywhere, which beats the letters merely
 * appearing in order. Every whitespace-separated term has to land somewhere,
 * so "new cust" finds "New customer" but not "New shipment".
 *
 * A field's weight scales everything it matches, so a hit on a record's name
 * outranks the same hit on its breadcrumb.
 */
export interface ScoredField {
  text: string;
  weight?: number;
  /**
   * Whether letters merely appearing in order count as a match. Worth it on a
   * short name ("wkrs" for Workers); on a long breadcrumb or description it
   * matches nearly anything, so secondary fields turn it off.
   */
  subsequence?: boolean;
}

const EXACT = 1;
const PREFIX = 0.9;
const WORD_PREFIX = 0.75;
const SUBSTRING = 0.5;
const SUBSEQUENCE = 0.2;
const MIN_SUBSEQUENCE_LENGTH = 3;
const WORD_BOUNDARY = /[\s\-_/.>·,()]+/;

function normalize(value: string): string {
  return value.normalize("NFKD").replace(/\p{M}/gu, "").toLowerCase().trim();
}

function isSubsequence(term: string, text: string): boolean {
  let cursor = 0;
  for (let i = 0; i < text.length && cursor < term.length; i++) {
    if (text[i] === term[cursor]) {
      cursor++;
    }
  }
  return cursor === term.length;
}

function scoreTerm(term: string, text: string, subsequence: boolean): number {
  if (text === term) {
    return EXACT;
  }
  if (text.startsWith(term)) {
    return PREFIX;
  }
  if (text.split(WORD_BOUNDARY).some((word) => word.startsWith(term))) {
    return WORD_PREFIX;
  }
  if (text.includes(term)) {
    return SUBSTRING;
  }
  if (subsequence && term.length >= MIN_SUBSEQUENCE_LENGTH && isSubsequence(term, text)) {
    return SUBSEQUENCE;
  }
  return 0;
}

/**
 * The best score any field gives the query, in (0, 1], or 0 when some term
 * matches nowhere. An empty query matches everything equally.
 */
export function fuzzyScore(query: string, fields: readonly ScoredField[]): number {
  const terms = normalize(query).split(/\s+/).filter(Boolean);
  if (terms.length === 0) {
    return 1;
  }

  const prepared = fields
    .filter((field) => field.text.length > 0)
    .map((field) => ({
      text: normalize(field.text),
      weight: field.weight ?? 1,
      subsequence: field.subsequence ?? true,
    }));
  if (prepared.length === 0) {
    return 0;
  }

  let total = 0;
  for (const term of terms) {
    let best = 0;
    for (const field of prepared) {
      const score = scoreTerm(term, field.text, field.subsequence) * field.weight;
      if (score > best) {
        best = score;
      }
    }
    if (best === 0) {
      return 0;
    }
    total += best;
  }

  return total / terms.length;
}

/**
 * Keeps the entries that match, best first. Ties keep their original order,
 * so a list that was already arranged on purpose stays arranged.
 */
export function rankByFuzzyScore<T>(
  query: string,
  entries: readonly T[],
  fieldsOf: (entry: T) => readonly ScoredField[],
): T[] {
  if (normalize(query) === "") {
    return [...entries];
  }

  return entries
    .map((entry, index) => ({ entry, index, score: fuzzyScore(query, fieldsOf(entry)) }))
    .filter((ranked) => ranked.score > 0)
    .sort((a, b) => b.score - a.score || a.index - b.index)
    .map((ranked) => ranked.entry);
}
