// source-edits.ts holds the edits the Studio makes to a source on the person's behalf, as
// CodeMirror changes, so each lands in the editor's undo history like typing would.

/**
 * The edit that appends a block to a document, after one blank line: it replaces the trailing
 * whitespace, so the rest of the text (and the editor's undo history for it) is untouched.
 */
export function appendBlock(
  source: string,
  block: string,
): { from: number; to: number; insert: string } {
  const from = source.replace(/\s+$/, "").length;
  return {
    from,
    to: source.length,
    insert: `${from > 0 ? "\n\n" : ""}${block.replace(/\s+$/, "")}\n`,
  };
}

/**
 * A new test block to fill in, named so it does not clash with the file's other tests: for a
 * tariff, one given and an expectation on the bill; for a library, an expectation to rewrite
 * over its functions.
 */
export function newTestBlock(
  kind: "tariff" | "library",
  existing: readonly string[],
  name: string,
): string {
  let title = name;
  for (let index = 2; existing.includes(title); index += 1) title = `${name} ${index}`;
  const quoted = title.replaceAll("\\", "").replaceAll('"', "'").replaceAll("{", "(").replaceAll("}", ")");
  const body =
    kind === "tariff"
      ? "  given shipment.distance = 100 mi\n  expect total > $0\n"
      : "  expect 1 + 1 == 2\n";
  return `test "${quoted}" {\n${body}}\n`;
}
