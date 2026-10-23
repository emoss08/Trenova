export type DiffLine = {
  kind: "same" | "add" | "del";
  text: string;
};

/** A run of unchanged lines a hunked diff leaves out. */
export type DiffGap = { kind: "gap"; count: number };

/**
 * A line diff of two sources by longest common subsequence: what a
 * proposal keeps, adds and removes, in the order a reader meets them. An
 * empty `before` is a new file, every line added.
 */
export function lineDiff(before: string, after: string): DiffLine[] {
  const a = before.trim() === "" ? [] : before.split("\n");
  const b = after.split("\n");
  const rows = a.length;
  const cols = b.length;
  const table: Uint16Array[] = Array.from({ length: rows + 1 }, () => new Uint16Array(cols + 1));

  for (let i = rows - 1; i >= 0; i--) {
    for (let j = cols - 1; j >= 0; j--) {
      table[i][j] =
        a[i] === b[j] ? table[i + 1][j + 1] + 1 : Math.max(table[i + 1][j], table[i][j + 1]);
    }
  }

  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < rows && j < cols) {
    if (a[i] === b[j]) {
      out.push({ kind: "same", text: a[i] });
      i++;
      j++;
    } else if (table[i + 1][j] >= table[i][j + 1]) {
      out.push({ kind: "del", text: a[i++] });
    } else {
      out.push({ kind: "add", text: b[j++] });
    }
  }
  while (i < rows) out.push({ kind: "del", text: a[i++] });
  while (j < cols) out.push({ kind: "add", text: b[j++] });

  return out;
}

/**
 * The diff as a reader scans it: every change with `context` unchanged lines on each side, and
 * each longer unchanged run folded into one gap that says how many lines it hides. A diff with
 * no change is returned whole.
 */
export function diffHunks(lines: DiffLine[], context = 2): (DiffLine | DiffGap)[] {
  const changed = lines.flatMap((line, index) => (line.kind === "same" ? [] : [index]));
  if (changed.length === 0) return lines;
  const keep = new Uint8Array(lines.length);
  for (const index of changed) {
    for (
      let near = Math.max(0, index - context);
      near <= Math.min(lines.length - 1, index + context);
      near++
    ) {
      keep[near] = 1;
    }
  }
  const out: (DiffLine | DiffGap)[] = [];
  let hidden = 0;
  for (let index = 0; index < lines.length; index++) {
    if (keep[index]) {
      if (hidden > 0) out.push({ kind: "gap", count: hidden });
      hidden = 0;
      out.push(lines[index]);
    } else {
      hidden++;
    }
  }
  if (hidden > 0) out.push({ kind: "gap", count: hidden });
  return out;
}
