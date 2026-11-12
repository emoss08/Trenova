/** The longest a gist runs before it is cut at a word. */
const GIST_MAX = 180;
/** A heading this near the start is the note's own lead-in, not what was found. */
const LEAD_IN_MAX = 200;

/**
 * What an agent wrote about its work, as one plain sentence a person can read
 * at a glance. An agent's note is a report in markdown, often several
 * paragraphs under headings ("What I found", "What I did"); the queue shows
 * what was found first and keeps the report for whoever asks for it.
 *
 * The lead-in up to the first heading is dropped when a heading opens the
 * note, the markdown marks are taken out, and the first sentence is kept.
 */
export function workNoteGist(note: string): string {
  let text = note.trim();
  if (text === "") {
    return "";
  }

  const heading = /\*\*[^*\n]{1,60}\*\*:?/.exec(text);
  if (heading && heading.index <= LEAD_IN_MAX) {
    const after = text.slice(heading.index + heading[0].length).trim();
    // A note that is one bold sentence and nothing after it is its own gist.
    text = after === "" ? heading[0] : after;
  }

  const plain = text
    .replace(/^#{1,6}\s+/gm, "")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/(^|\s)[*_]([^*_\n]+)[*_](?=\s|[.,;:!?]|$)/g, "$1$2")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/(^|\s)[-*•]\s+/g, "$1")
    .replace(/\s+/g, " ")
    .trim();

  const sentence = /^.*?[.!?](?=\s|$)/.exec(plain)?.[0] ?? plain;
  if (sentence.length <= GIST_MAX) {
    return sentence;
  }
  const cut = sentence.slice(0, GIST_MAX);

  return `${cut.slice(0, Math.max(cut.lastIndexOf(" "), GIST_MAX / 2)).trimEnd()}…`;
}

/** Whether the note says more than its gist, so there is something to open. */
export function workNoteHasMore(note: string): boolean {
  const gist = workNoteGist(note);

  return (
    gist !== "" &&
    note
      .replace(/[*_`#>-]/g, "")
      .replace(/\s+/g, " ")
      .trim().length >
      gist.length + 8
  );
}
