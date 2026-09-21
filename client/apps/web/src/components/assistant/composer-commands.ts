import type { Suggestion } from "./suggestions";

/**
 * A draft that opens with a slash is a request for the starter questions,
 * filtered by whatever follows. Null for any other draft, including a slash
 * somewhere later in a sentence and a draft that has grown to a second line.
 */
export function slashQuery(draft: string): string | null {
  if (!draft.startsWith("/") || draft.includes("\n")) {
    return null;
  }

  return draft.slice(1).trim().toLowerCase();
}

/**
 * The starter questions matching a slash query, on their label or their
 * prompt. An empty query is every one of them.
 */
export function matchSuggestions(query: string, suggestions: readonly Suggestion[]): Suggestion[] {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return [...suggestions];
  }

  return suggestions.filter(
    (suggestion) =>
      suggestion.label.toLowerCase().includes(needle) ||
      suggestion.prompt.toLowerCase().includes(needle),
  );
}
