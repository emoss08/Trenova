// tariff-source-lines.ts places a position of a Tariff source on its line, for the "Ln 12" a
// problem, a test, a reference or an unreached region shows beside it.

/** The 1-based line a UTF-16 position falls on. */
export function lineOf(source: string, offset: number): number {
  let line = 1;
  const end = Math.min(offset, source.length);
  for (let index = 0; index < end; index += 1) if (source.charCodeAt(index) === 10) line += 1;
  return line;
}
