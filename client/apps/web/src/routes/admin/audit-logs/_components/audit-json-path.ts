/**
 * buildJsonPathIndex maps each line of JSON written with `JSON.stringify(value, null, 2)` that
 * holds a value or opens one to its JSONPath, so clicking a line can copy where that value lives.
 * It reads the text back rather than the value it came from, so what the serializer dropped (an
 * undefined property) or rewrote (a Date) cannot shift a line. It walks the layout the serializer
 * writes: an object or array opens and closes on lines of its own, except an empty one, which is
 * written whole on one line ("{}", "[]").
 */
export function buildJsonPathIndex(json: string): Map<number, string> {
  const map = new Map<number, string>();
  let line = 0;

  function walk(val: object, path: string) {
    const entries: Array<[string, unknown]> = Array.isArray(val)
      ? val.map((child, i) => [`${path}[${i}]`, child])
      : Object.entries(val).map(([key, child]) => [`${path}.${key}`, child]);

    if (entries.length === 0) {
      line++;
      return;
    }

    line++;
    for (const [childPath, child] of entries) {
      map.set(line, childPath);
      if (child !== null && typeof child === "object") {
        walk(child, childPath);
      } else {
        line++;
      }
    }
    line++;
  }

  let value: unknown;
  try {
    value = JSON.parse(json);
  } catch {
    return map;
  }
  if (value !== null && typeof value === "object") {
    walk(value, "$");
  }
  return map;
}
