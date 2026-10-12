const ESCAPES: Record<string, string> = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#39;",
};

function escapeHtml(text: string): string {
  return text.replace(/[&<>"']/g, (character) => ESCAPES[character] ?? character);
}

/** The value as it would be searched for on a page: spacing folded, case ignored. */
export function normalizeNeedle(value: string): string {
  return value.replace(/\s+/g, " ").trim().toLowerCase();
}

/**
 * One run of a PDF's text layer as HTML, with every place the value Nova
 * read appears wrapped in a mark, so hovering a field shows where on the
 * page it came from. The text is escaped: it is the document's, not ours.
 */
export function markMatches(text: string, needle: string): string {
  const wanted = normalizeNeedle(needle);
  if (wanted.length < 2) {
    return escapeHtml(text);
  }
  const lower = text.toLowerCase();
  let html = "";
  let from = 0;
  let at = lower.indexOf(wanted, from);
  while (at !== -1) {
    html += `${escapeHtml(text.slice(from, at))}<mark>${escapeHtml(text.slice(at, at + wanted.length))}</mark>`;
    from = at + wanted.length;
    at = lower.indexOf(wanted, from);
  }

  return html + escapeHtml(text.slice(from));
}
