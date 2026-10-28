/**
 * The chapter titles that have typed themselves in on this page.
 *
 * A heading types a fresh title in when it first shows. The same heading is
 * drawn twice over a turn's life: under the live turn while it runs, then
 * among the saved messages once it is saved, and that is a new component,
 * moments after the model named the chapter. Deciding from the title's age
 * alone typed it in again at every remount, so a title looked as if it were
 * being written afresh each time the conversation settled. A title types in
 * once per page; after that it shows whole.
 */
const streamed = new Set<string>();

/** Enough for every chapter of the conversations one page session opens. */
const MAX_REMEMBERED = 2000;

export function chapterTitleKey(chapterId: string, title: string): string {
  return `${chapterId}:${title}`;
}

export function hasStreamed(key: string): boolean {
  return streamed.has(key);
}

export function markStreamed(key: string): void {
  if (streamed.has(key)) {
    return;
  }
  if (streamed.size >= MAX_REMEMBERED) {
    const oldest = streamed.values().next();
    if (!oldest.done) {
      streamed.delete(oldest.value);
    }
  }
  streamed.add(key);
}

/** Forgets every title, for tests. */
export function resetStreamedTitles(): void {
  streamed.clear();
}
