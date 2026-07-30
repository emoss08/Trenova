import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, resolve } from "node:path";

// A test rather than a lint rule, deliberately.
//
// oxlint is the linter here and has no custom-rule plugin API in this configuration;
// getting a bespoke rule would mean reinstating ESLint over the same files purely for this
// one check, and maintaining two linters. This runs inside the existing test job, can state
// the predicate exactly, and fails on the seventy-fifth call site the same way it fails on
// the first.
//
// What it guards: handleMutationError needs a real react-hook-form instance to tell a
// registered leaf from an unregistered path. MutationFormTarget asks only for
// { control, setError } because react-hook-form's Control carries an invariant generic that
// makes UseFormReturn unassignable across forms declared with an output type — so the type
// cannot reject a hand-rolled stub, and nothing stops a caller from passing one.

const SRC = resolve(__dirname, "../..");
const MUTATION_ENTRYPOINTS = ["useApiMutation", "useOptimisticMutation", "handleMutationError"];

// These two declare the contract and forward `form` onward; they are not call sites. The
// legacy-prop scan still covers them.
const DEFINITION_FILES = ["hooks/use-api-mutation.ts", "hooks/use-optimistic-mutation.ts"];

// A form identifier is trustworthy if the file binds it from react-hook-form or receives it
// as a prop typed by react-hook-form / the transport's own contract type.
const TRUSTWORTHY_BINDING = [
  "useForm(",
  "useForm<",
  "useFormContext(",
  "useFormContext<",
  "UseFormReturn",
  "MutationFormTarget",
];

function sourceFiles(dir: string, found: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === "node_modules" || entry === "__tests__") {
      continue;
    }
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) {
      sourceFiles(full, found);
    } else if (/\.tsx?$/.test(full) && !/\.(test|spec|stories)\.tsx?$/.test(full)) {
      found.push(full);
    }
  }
  return found;
}

type CallSite = { file: string; value: string };

export function collectFormCallSites(root: string) {
  const passed: CallSite[] = [];
  const legacy: CallSite[] = [];
  const untrustworthy: CallSite[] = [];

  for (const file of sourceFiles(root)) {
    const source = readFileSync(file, "utf8");
    if (!MUTATION_ENTRYPOINTS.some((name) => source.includes(name))) {
      continue;
    }

    const rel = relative(root, file);

    // The prop this migration removed. Its presence means a call site still hands over a
    // bare setError, which cannot resolve registered leaves.
    for (const _match of source.matchAll(/\bsetFormError\b/g)) {
      legacy.push({ file: rel, value: "setFormError" });
    }

    if (DEFINITION_FILES.includes(rel)) {
      continue;
    }

    // `form,` shorthand or `form: <expr>,` inside an options object.
    for (const match of source.matchAll(/^\s{2,}form(?:,|:\s*([^\n]+?),)\s*$/gm)) {
      const value = (match[1] ?? "form").trim();
      passed.push({ file: rel, value });

      const isIdentifier = /^[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)*$/.test(value);
      if (!isIdentifier || !TRUSTWORTHY_BINDING.some((token) => source.includes(token))) {
        untrustworthy.push({ file: rel, value });
      }
    }
  }

  return { passed, legacy, untrustworthy };
}

describe("form call sites", () => {
  const result = collectFormCallSites(SRC);

  // Guards against the scan silently matching nothing and passing vacuously.
  it("finds the migrated call sites", () => {
    expect(result.passed.length).toBeGreaterThanOrEqual(74);
  });

  it("has no call site still passing a bare setError", () => {
    expect(result.legacy).toEqual([]);
  });

  it("passes a form derived from react-hook-form at every call site", () => {
    expect(result.untrustworthy).toEqual([]);
  });

  it("never passes an object literal as the form", () => {
    const literals = result.passed.filter((site) => site.value.startsWith("{"));
    expect(literals).toEqual([]);
  });
});
