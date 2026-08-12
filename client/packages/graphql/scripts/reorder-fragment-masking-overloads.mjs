import { readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

// client-preset emits the Partial overload of getFragmentData first, so it wins
// resolution for every call and degrades ordinary fragment data to Partial.
// Moving it last keeps it available for conditional directives while letting the
// precise overloads match first.
const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const target = resolve(packageRoot, "src/generated/fragment-masking.ts");

const partialOverload =
  /\/\/ return partial if `fragmentType` is partial e\.g\. because of conditional directives\nexport function getFragmentData<TType>\(\n {2}_documentNode: DocumentTypeDecoration<TType, any>,\n {2}fragmentType: FragmentType<DocumentTypeDecoration<Partial<TType>, any>>\n\): Partial<TType>;\n/;

const source = await readFile(target, "utf8");
const match = source.match(partialOverload);

if (!match) {
  if (!source.includes("): Partial<TType>;")) {
    throw new Error(
      `Expected the Partial overload of getFragmentData in ${target}. ` +
        "The client-preset output changed shape — review it and update this script.",
    );
  }
  process.exit(0);
}

const withoutPartial = source.replace(partialOverload, "");
const lastOverloadEnd = withoutPartial.lastIndexOf(";\nexport function getFragmentData<TType>(");

if (lastOverloadEnd === -1) {
  throw new Error(
    `Could not locate the getFragmentData implementation signature in ${target}.`,
  );
}

const insertAt = lastOverloadEnd + 2;
const reordered =
  withoutPartial.slice(0, insertAt) + match[0] + withoutPartial.slice(insertAt);

if (reordered !== source) {
  await writeFile(target, reordered, "utf8");
}
