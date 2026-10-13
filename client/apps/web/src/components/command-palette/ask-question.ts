const QUESTION_OPENERS =
  /^(how|what|whats|what's|why|which|who|whos|who's|when|where|is|are|did|does|do|can|could|should|will|show me|list|find me|tell me|summarize|draft|any)\b/i;
const MIN_QUESTION_WORDS = 3;

/**
 * A palette entry that is a question rather than a search: it ends with a
 * question mark, or it opens the way a question does and runs to three words
 * or more. "where" alone is a search; "where is S-240193" is a question.
 */
export function askQuestion(value: string): string | null {
  const trimmed = value.trim();
  if (trimmed === "" || trimmed === "?") {
    return null;
  }
  if (trimmed.endsWith("?")) {
    return trimmed;
  }
  if (QUESTION_OPENERS.test(trimmed) && trimmed.split(/\s+/).length >= MIN_QUESTION_WORDS) {
    return trimmed;
  }
  return null;
}

const QUESTION_FRAMING = new Set([
  "a",
  "about",
  "an",
  "and",
  "any",
  "anything",
  "are",
  "at",
  "can",
  "could",
  "did",
  "do",
  "does",
  "draft",
  "find",
  "for",
  "from",
  "get",
  "give",
  "going",
  "happened",
  "happening",
  "how",
  "i",
  "in",
  "is",
  "it",
  "its",
  "know",
  "list",
  "me",
  "my",
  "of",
  "on",
  "or",
  "our",
  "please",
  "should",
  "show",
  "summarize",
  "tell",
  "that",
  "the",
  "their",
  "there",
  "this",
  "to",
  "up",
  "us",
  "was",
  "we",
  "were",
  "what",
  "whats",
  "when",
  "where",
  "which",
  "who",
  "whos",
  "why",
  "will",
  "with",
  "would",
]);
const TOKEN_EDGES = /^[^\p{L}\p{N}#]+|[^\p{L}\p{N}]+$/gu;
const POSSESSIVE = /['’]s$/iu;
const APOSTROPHES = /['’]/gu;

/**
 * The words of a question worth searching records for: what is left once the
 * way it is asked is taken away. The index matches every word it is given and
 * drops words from the end until something matches, so "Tell me about
 * SEED-PAY-002?" would give up the one word that names a record before it gave
 * up "tell"; "SEED-PAY-002" finds the shipment.
 */
export function questionSearchText(question: string): string {
  return question
    .split(/\s+/)
    .map((word) => word.replace(TOKEN_EDGES, "").replace(POSSESSIVE, ""))
    .filter(
      (word) => word !== "" && !QUESTION_FRAMING.has(word.toLowerCase().replace(APOSTROPHES, "")),
    )
    .join(" ");
}

/** The palette runs commands only, while what is typed starts with ">". */
export function commandModeQuery(value: string): string | null {
  return value.startsWith(">") ? value.slice(1) : null;
}
