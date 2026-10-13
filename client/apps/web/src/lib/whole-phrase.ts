const WORD_CHARACTER = /[\p{L}\p{N}]/u;
const CURLY_APOSTROPHES = /[‘’]/gu;
const WHITESPACE_RUN = /\s+/gu;

/** Text as a person reads it for a match: any case, any apostrophe, any run of spaces. */
export function foldPhrase(value: string): string {
  return value
    .toLocaleLowerCase()
    .replace(CURLY_APOSTROPHES, "'")
    .replace(WHITESPACE_RUN, " ")
    .trim();
}

/**
 * Whether text contains phrase as whole words, folded alike: "SEED-PAY-002"
 * is in "Tell me about seed-pay-002?", "Acme" is in "Acme's invoices", and
 * "Acme" is not in "Acmeville".
 */
export function containsWholePhrase(text: string, phrase: string): boolean {
  const haystack = foldPhrase(text);
  const needle = foldPhrase(phrase);
  if (needle === "") {
    return false;
  }
  for (let at = haystack.indexOf(needle); at !== -1; at = haystack.indexOf(needle, at + 1)) {
    const before = haystack.charAt(at - 1);
    const after = haystack.charAt(at + needle.length);
    if (!WORD_CHARACTER.test(before) && !WORD_CHARACTER.test(after)) {
      return true;
    }
  }
  return false;
}
