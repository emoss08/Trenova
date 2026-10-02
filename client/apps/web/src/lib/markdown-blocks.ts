const FENCE_OPEN = /^ {0,3}(`{3,}|~{3,})/;
const LIST_ITEM = /^(?:[-*+]|\d{1,9}[.)])(?:[ \t]|$)/;
const LIST_ITEM_SO_FAR = /^(?:[-*+]|\d{1,9}[.)]?)$/;

type Fence = { marker: string; length: number };

function opensFence(line: string): Fence | null {
  const match = FENCE_OPEN.exec(line);
  if (!match) {
    return null;
  }
  const run = match[1];

  return { marker: run[0], length: run.length };
}

function closesFence(line: string, fence: Fence): boolean {
  const trimmed = line.replace(/^ {0,3}/, "").trimEnd();
  if (trimmed.length < fence.length) {
    return false;
  }
  for (const char of trimmed) {
    if (char !== fence.marker) {
      return false;
    }
  }

  return true;
}

/**
 * Whether a line after a blank one begins a block of its own. A list item
 * does not: two lists split at the blank line between their items render as
 * two lists. Neither does an indented line, which continues what came before
 * it. The last line of text that is still arriving is judged by what it could
 * yet become, so a "1" that turns into "1. " never moves a boundary that has
 * already been drawn.
 */
function beginsBlock(line: string, complete: boolean): boolean {
  if (line === "" || line[0] === " " || line[0] === "\t") {
    return false;
  }
  if (LIST_ITEM.test(line)) {
    return false;
  }

  return complete || !LIST_ITEM_SO_FAR.test(line);
}

/**
 * Cuts markdown into its top-level blocks, at the blank lines between them and
 * never inside a fence, so the blocks joined are the text exactly.
 *
 * A reply that is still streaming is re-rendered on every token; parsing it
 * whole each time costs the square of its length. Cut into blocks, every block
 * but the last is final once a later one has begun, and only the last is
 * parsed again.
 */
export function splitMarkdownBlocks(markdown: string): string[] {
  const blocks: string[] = [];
  let start = 0;
  let offset = 0;
  let fence: Fence | null = null;
  let afterBlank = false;

  while (offset < markdown.length) {
    const newline = markdown.indexOf("\n", offset);
    const complete = newline !== -1;
    const end = complete ? newline + 1 : markdown.length;
    const line = markdown.slice(offset, complete ? newline : end);

    if (fence) {
      if (closesFence(line, fence)) {
        fence = null;
      }
      afterBlank = false;
    } else if (line.trim() === "") {
      afterBlank = true;
    } else {
      if (afterBlank && offset > start && beginsBlock(line, complete)) {
        blocks.push(markdown.slice(start, offset));
        start = offset;
      }
      fence = opensFence(line);
      afterBlank = false;
    }

    offset = end;
  }

  if (start < markdown.length) {
    blocks.push(markdown.slice(start));
  }

  return blocks;
}
