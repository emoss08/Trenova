/**
 * A palette entry that is a question rather than a search: it ends with a
 * question mark, or opens with a chevron for people who prefer a prefix.
 */
export function askQuestion(value: string): string | null {
  const trimmed = value.trim();
  if (trimmed.startsWith(">")) {
    const question = trimmed.slice(1).trim();

    return question === "" ? null : question;
  }
  if (trimmed.endsWith("?") && trimmed.length > 3) {
    return trimmed;
  }

  return null;
}
