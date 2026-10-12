/** A run of pages filed as one document of one type. */
export type PageRange = { fromPage: number; toPage: number; documentTypeId: string };

/**
 * The documents a split makes: each run of neighbouring pages given the same
 * type becomes one document, and a page marked not to file (null) is left
 * out and breaks the run. `types[i]` is the type chosen for `pages[i]`.
 */
export function splitRanges(
  pages: readonly number[],
  types: readonly (string | null)[],
): PageRange[] {
  const ranges: PageRange[] = [];
  pages.forEach((page, index) => {
    const type = types[index] ?? null;
    if (type === null) {
      return;
    }
    const last = ranges.at(-1);
    const previousPage = pages[index - 1];
    const joins =
      last !== undefined &&
      last.documentTypeId === type &&
      last.toPage === page - 1 &&
      previousPage === page - 1 &&
      (types[index - 1] ?? null) === type;
    if (joins) {
      last.toPage = page;
    } else {
      ranges.push({ fromPage: page, toPage: page, documentTypeId: type });
    }
  });

  return ranges;
}
