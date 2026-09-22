import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";

/**
 * Reading an enum out of the Go source, for the client enums that have to
 * cover it.
 *
 * A hand-listed zod enum over a server-side set is not a validation, it is a
 * copy that silently falls behind. When the copy is short the server's
 * response fails to parse and the whole payload is discarded — not the one
 * unknown row, the payload — so a page goes blank over a value somebody added
 * in Go three weeks ago. A test that lists the values it expects is a second
 * copy with the same problem, and it passes while the page is broken.
 *
 * These read the server's own list instead, so the test fails when the client
 * falls behind rather than when somebody remembers to update it.
 */

/** The nearest ancestor holding both halves of the monorepo. */
export function repoRoot(): string {
  let current = process.cwd();
  for (let depth = 0; depth < 8; depth++) {
    try {
      readFileSync(join(current, "go.work"), "utf8");

      return current;
    } catch {
      current = dirname(current);
    }
  }

  throw new Error("could not find the repository root from " + process.cwd());
}

type GoEnumOptions = {
  /** Path to the Go file, relative to the repository root. */
  file: string;
  /** The Go type, e.g. "Task" or "Template". */
  typeName: string;
  /** The function listing every member; defaults to All{typeName}s. */
  listFn?: string;
};

/**
 * The wire values of a Go string enum, in the order its All…() lists them.
 *
 * The list function is the source rather than the constant block, because a
 * constant that nothing lists is one the server never serves — and because
 * that list is what the API hands the client.
 */
export function goEnumValues({ file, typeName, listFn }: GoEnumOptions): string[] {
  const source = readFileSync(join(repoRoot(), file), "utf8");
  const listName = listFn ?? `All${typeName}s`;
  const listPattern =
    `func ${listName}\\(\\) \\[\\]${typeName} \\{\\s*` +
    `return \\[\\]${typeName}\\{([\\s\\S]*?)\\n\\t\\}`;
  const block = new RegExp(listPattern).exec(source);
  if (block === null) {
    throw new Error(`could not find ${listName}() in ${file}`);
  }

  const names = [...block[1].matchAll(new RegExp(`${typeName}([A-Za-z]+),`, "g"))].map(
    (match) => match[1],
  );
  // Both forms Go writes a string enum in: the conversion
  // (`TaskFoo = Task("foo")`) and the typed literal (`KindFoo Kind = "foo"`),
  // with the repeated type optional because a const block only needs it on
  // the first line.
  const constantPattern =
    `${typeName}([A-Za-z]+)\\s*(?:${typeName}\\s*)?=\\s*` +
    `(?:${typeName}\\()?"([^"]+)"\\)?`;
  const constants = new Map(
    [...source.matchAll(new RegExp(constantPattern, "g"))].map((match) => [match[1], match[2]]),
  );

  return names.map((name) => {
    const value = constants.get(name);
    if (value === undefined) {
      throw new Error(`${listName}() names ${typeName}${name}, which has no constant`);
    }

    return value;
  });
}
