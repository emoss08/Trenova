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

/** The palette runs commands only, while what is typed starts with ">". */
export function commandModeQuery(value: string): string | null {
  return value.startsWith(">") ? value.slice(1) : null;
}
