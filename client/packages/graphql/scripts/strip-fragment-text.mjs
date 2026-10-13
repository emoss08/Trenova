// The client preset replaces every operation with its hash, because the server only accepts
// persisted operations, but leaves each fragment as a TypedDocumentString holding the
// fragment's GraphQL text. Nothing at runtime reads that text: getFragmentData and
// makeFragmentData ignore the document, isFragmentReady reads only __meta__.fragmentName,
// and an operation's text reaches the server as its hash. Left in, it shipped about 150 kB
// of GraphQL source to every browser. This rewrites each fragment to the same metadata-only
// object an operation becomes, keeping its type.

import { readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const target = resolve(dirname(fileURLToPath(import.meta.url)), "../src/generated/graphql.ts");

const FRAGMENT_DOCUMENT =
  /new TypedDocumentString\(`[^`]*`, (\{"fragmentName":"[A-Za-z_][A-Za-z0-9_]*"\})\) as unknown as TypedDocumentString</g;

export function stripFragmentText(source) {
  let replaced = 0;
  const out = source.replace(FRAGMENT_DOCUMENT, (_match, meta) => {
    replaced += 1;
    return `{"__meta__":${meta}} as unknown as TypedDocumentString<`;
  });
  const remaining = out.match(/new TypedDocumentString\(`/g)?.length ?? 0;
  if (remaining > 0) {
    throw new Error(
      `strip-fragment-text: ${remaining} document(s) still carry GraphQL text after rewriting ` +
        `${replaced}; the generator's output changed shape. Update FRAGMENT_DOCUMENT.`,
    );
  }
  return out;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const source = await readFile(target, "utf8");
  const stripped = stripFragmentText(source);
  if (stripped !== source) await writeFile(target, stripped);
}
