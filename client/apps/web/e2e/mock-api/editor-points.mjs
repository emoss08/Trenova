// Where to point in a CodeMirror editor, for the screenshot walkthroughs of the Formula Studio
// and the Lading studio.

/**
 * Where the character `delta` past the first `text` is drawn in the editor. A line's text
 * includes its inlay hints, so a needle is written as the line reads on screen.
 */
export async function pointAt(page, text, delta = 0) {
  const point = await page.evaluate(
    ({ needle, shift }) => {
      for (const line of document.querySelectorAll(".cm-content .cm-line")) {
        const start = (line.textContent ?? "").indexOf(needle.split("\n")[0]);
        if (start < 0) continue;
        let remaining = start + shift;
        const walker = document.createTreeWalker(line, NodeFilter.SHOW_TEXT);
        for (let node = walker.nextNode(); node; node = walker.nextNode()) {
          const length = node.textContent?.length ?? 0;
          if (remaining <= length) {
            const range = document.createRange();
            range.setStart(node, remaining);
            range.setEnd(node, Math.min(remaining + 1, length));
            const rect = range.getBoundingClientRect();
            return { x: rect.left + 1, y: rect.top + rect.height / 2 };
          }
          remaining -= length;
        }
      }
      return null;
    },
    { needle: text, shift: delta },
  );
  if (!point) throw new Error(`no ${text} in the editor`);
  return point;
}

/** Clicks the editor just before the character `delta` past the first `text` on a line. */
export async function cursorAt(page, text, delta = 0) {
  const point = await pointAt(page, text, delta);
  await page.mouse.click(point.x, point.y);
}
