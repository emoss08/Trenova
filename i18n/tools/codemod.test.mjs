// Run with: node --test i18n/tools/
//
// The codemod rewrites ~1,500 files, so its edge cases are pinned here rather than
// discovered in a diff. Each case is one thing that would be wrong in the product if the
// rule changed: a hook in a non-component breaks the rules of hooks, a swallowed space
// runs words into an icon, a frozen module-level call pins the language at import.
import assert from "node:assert/strict";
import { test } from "node:test";
import { transformSource } from "./codemod.mjs";

const run = (src) => transformSource(src, "sample.tsx");

test("wraps jsx text and adds the hook inside a component", () => {
  const out = run(`export function Panel() {\n  return <h2>Create Shipment</h2>;\n}\n`).output;
  assert.match(out, /const t = useT\(\);/);
  assert.match(out, /\{t\("Create Shipment"\)\}/);
  assert.match(out, /i18n\/use-t/);
});

test("wraps allowlisted props and leaves identifier props alone", () => {
  const out = run(
    `export function Panel() {\n  return <Field label="Save" name="save" variant="ghost" />;\n}\n`,
  ).output;
  assert.match(out, /label=\{t\("Save"\)\}/);
  assert.match(out, /name="save"/, "name is a form key, not a caption");
  assert.match(out, /variant="ghost"/);
});

test("never wraps an svg path", () => {
  const result = run(`export function Icon() {\n  return <path d="M12 2L2 7" />;\n}\n`);
  assert.equal(result.changed, false);
});

test("folds a sentence broken by an interpolation", () => {
  const out = run(
    `export function Panel({ name }: any) {\n  return <p>Delete "{name}"? This cannot be undone.</p>;\n}\n`,
  ).output;
  assert.match(out, /t\("Delete \\"\{0\}\\"\? This cannot be undone\.", name\)/);
});

test("folds across a sibling element and keeps the separating space", () => {
  const out = run(
    `export function Panel({ p }: any) {\n  return <a><Logo /> Continue with {p.name}</a>;\n}\n`,
  ).output;
  assert.match(out, /<Logo \/> \{t\("Continue with \{0\}", p\.name\)\}/);
});

test("leaves a run of interpolations with no words alone", () => {
  const result = run(`export function Panel({ a, b }: any) {\n  return <span>{a} {b}</span>;\n}\n`);
  assert.equal(result.changed, false);
});

test("resolves a toast inside a nested handler to the component's hook", () => {
  const out = run(
    `import { toast } from "sonner";\nexport function Save() {\n  const go = () => { toast.success("Settings updated"); };\n  return <button onClick={go}>Go</button>;\n}\n`,
  ).output;
  assert.match(out, /toast\.success\(t\("Settings updated"\)\)/);
  assert.equal(out.match(/const t = useT\(\);/g).length, 1, "one binding per component");
});

test("uses the module-level translate outside a component", () => {
  const out = run(
    `export function describe() {\n  return <span>Pending review</span>;\n}\n`,
  ).output;
  assert.match(out, /translate\("Pending review"\)/);
  assert.match(out, /i18n\/runtime/);
  assert.doesNotMatch(out, /useT\(\)/, "a lowercase helper is not a component");
});

test("treats a custom hook as a hook context", () => {
  const out = run(
    `export function useColumns() {\n  return [<span key="a">Status</span>];\n}\n`,
  ).output;
  assert.match(out, /const t = useT\(\);/);
});

test("does not add a second hook binding to a file that already has one", () => {
  const out = run(
    `import { useT } from "@trenova/shared/i18n/use-t";\nexport function Panel() {\n  const t = useT();\n  return <h2>Create Shipment</h2>;\n}\n`,
  ).output;
  assert.equal(out.match(/const t = useT\(\);/g).length, 1);
  assert.equal(out.match(/i18n\/use-t/g).length, 1);
});

test("leaves object label maps to be translated at render", () => {
  const result = run(`export const NAV = [{ label: "Shipments", href: "/s" }];\n`);
  assert.equal(
    result.changed,
    false,
    "a module-level const evaluates once, so wrapping here would freeze the language",
  );
});

test("reports a parse failure instead of writing a broken file", () => {
  const result = run(`export function Broken( {\n`);
  assert.equal(result.changed, false);
  assert.equal(result.skipped[0].reason, "parse error");
});

test("gives a concise arrow component a body instead of skipping it", () => {
  const out = run(`const Ring = ({ size }: any) => (\n  <svg>\n    <title>Loading...</title>\n  </svg>\n);\n`).output;
  assert.match(out, /const t = useT\(\);/);
  assert.match(out, /return \(/);
  assert.match(out, /\{t\("Loading\.\.\."\)\}/);
});

test("handles a single-line concise arrow", () => {
  const out = run(`const Tiny = () => <b>Saved</b>;\n`).output;
  assert.match(out, /const Tiny = \(\) => \{\n  const t = useT\(\);\n  return <b>\{t\("Saved"\)\}<\/b>;\n\};/);
});

test("never folds an expression that produces markup", () => {
  const out = run(
    `export function P({ run }: any) {\n  return <span>Run Console\n  {run && (<Badge>Live</Badge>)}</span>;\n}\n`,
  ).output;
  assert.match(out, /\{t\("Run Console"\)\}/, "the text is wrapped on its own");
  assert.match(out, /<Badge>\{t\("Live"\)\}<\/Badge>/, "and the nested text keeps its own entry");
  assert.doesNotMatch(out, /t\("Run Console \{0\}"/, "an element cannot be a string placeholder");
});

test("keeps a parenthesised argument balanced", () => {
  // Babel excludes wrapping parentheses from a node's range, so slicing by the expression
  // would emit the unbalanced `a && (b + 1`.
  const out = run(
    `export function P({ a, b }: any) {\n  return <i>Total {a && (b + 1)} loads</i>;\n}\n`,
  ).output;
  assert.match(out, /t\("Total \{0\} loads", a && \(b \+ 1\)\)/);
});

test("adds t to the dependency array of a hook it edited", () => {
  const out = run(
    `export function P({ x }: any) {\n  const f = useCallback(() => toast.success("Saved"), [x]);\n  return <div onClick={f} />;\n}\n`,
  ).output;
  assert.match(out, /\[x, t\]/, "a memoized callback must rebuild when the language changes");
});

test("adds t to an empty dependency array", () => {
  const out = run(
    `export function P() {\n  const l = useMemo(() => <b>Total</b>, []);\n  return <div>{l}</div>;\n}\n`,
  ).output;
  assert.match(out, /\[t\]/);
});

test("leaves an untouched hook's dependencies alone", () => {
  const out = run(
    `export function P() {\n  const g = useMemo(() => [1], []);\n  return <div>Total{g}</div>;\n}\n`,
  ).output;
  assert.match(out, /useMemo\(\(\) => \[1\], \[\]\)/);
});

test("keeps a directive prologue first", () => {
  // "use no memo" only works as the first statement; inserting above it turns the React
  // Compiler opt-out it encodes into a bare string expression.
  const out = run(`export function P() {\n  "use no memo";\n  return <b>Total</b>;\n}\n`).output;
  assert.match(out, /\{\n  "use no memo";\n  const t = useT\(\);/);
});

test("does not create a sparse array when deps end with a trailing comma", () => {
  const out = run(
    `export function Q({ x }: any) {\n  const f = useCallback(() => toast.success("Saved"), [\n    x,\n  ]);\n  return <div onClick={f} />;\n}\n`,
  ).output;
  assert.doesNotMatch(out, /,\s*,/, "[x, , t] has a hole");
  assert.match(out, /x,\s*t\]/);
});
