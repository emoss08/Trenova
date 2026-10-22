export type DiffLine = {
  kind: "same" | "add" | "del";
  text: string;
};

/**
 * A line diff of two expressions by longest common subsequence: what a
 * proposal keeps, adds and removes, in the order a reader meets them. An
 * empty `before` is a new formula, every line added.
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
