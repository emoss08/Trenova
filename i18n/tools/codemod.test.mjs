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

test("keeps the space between two interpolations", () => {
  // `{years} {unit} on {date}` must not fold to "{0}{1} on {2}" — that renders "2years on".
  const out = run(
    `export function A({ item }: any) {\n  return <span>{item.years} {item.years === 1 ? "year" : "years"} on{" "}{fmt(item.onDate)}</span>;\n}\n`,
  ).output;
  assert.match(out, /t\("\{0\} \{1\} on \{2\}"/);
});

test("still treats whitespace between elements as layout", () => {
  const result = run(`export function B() {\n  return <div>\n  <A />\n  <B />\n</div>;\n}\n`);
  assert.equal(result.changed, false);
});

test("labels mode translates label maps at render, not at definition", () => {
  const out = transformSource(
    `export function Nav({ items }: any) {\n  return <ul>{items.map((item: any) => (<li key={item.label}><span>{item.label}</span></li>))}</ul>;\n}\n`,
    "s.tsx",
    { labels: true },
  ).output;
  assert.match(out, /<span>\{t\(item\.label\)\}<\/span>/);
  assert.match(out, /key=\{item\.label\}/, "a key is an identity, not a caption");
});

test("labels mode is off by default", () => {
  const result = transformSource(
    `export function Nav({ item }: any) {\n  return <span>{item.label}</span>;\n}\n`,
    "s.tsx",
  );
  assert.equal(result.changed, false);
});

test("adds a binding to a second component in an already-migrated file", () => {
  // Whether `t` is bound is a property of the function, not of the file. A file migrated in
  // an earlier pass can still contain a component that needs its own binding.
  const out = transformSource(
    `import { useT } from "@trenova/shared/i18n/use-t";\n` +
      `export function First() {\n  const t = useT();\n  return <b>{t("Save")}</b>;\n}\n` +
      `export function Second({ item }: any) {\n  return <b>{item.label}</b>;\n}\n`,
    "s.tsx",
    { labels: true },
  ).output;
  assert.equal(out.match(/const t = useT\(\);/g).length, 2);
  assert.equal(out.match(/i18n\/use-t/g).length, 1, "the import is per file");
});

test("labels mode translates a caption passed as a prop, but never a key", () => {
  const out = transformSource(
    `export function F({ option, group }: any) {\n  return <Cmd key={group.label} heading={group.label}><Item label={option.label} value={option.id} /></Cmd>;\n}\n`,
    "s.tsx",
    { labels: true },
  ).output;
  assert.match(out, /heading=\{t\(group\.label\)\}/);
  assert.match(out, /label=\{t\(option\.label\)\}/);
  assert.match(out, /key=\{group\.label\}/);
  assert.match(out, /value=\{option\.id\}/);
});

test("labels mode does not wrap an already-wrapped caption", () => {
  const out = transformSource(
    `export function F({ option }: any) {\n  const t = useT();\n  return <Item label={t(option.label)} />;\n}\n`,
    "s.tsx",
    { labels: true },
  );
  assert.equal(out.changed, false);
});

// Rendered literals. A string that is displayed but sits in an expression rather than in
// JSX text was the codemod's blind spot: the extractor never saw it, so it was translated
// nowhere and rendered English inside otherwise translated screens.

test("wraps a literal rendered from a ternary in jsx child position", () => {
  const out = run(
    `export function Panel({ filed }: any) {\n  return <span>{filed ? "Regenerate" : "Generate"}</span>;\n}\n`,
  ).output;
  assert.match(out, /\{filed \? t\("Regenerate"\) : t\("Generate"\)\}/);
});

test("wraps a placeholder value handed into an existing t() call", () => {
  const out = run(
    `export function Panel({ on }: any) {\n  const t = useT();\n  return <span>{t("Auto-match: {0}", on ? "On" : "Off")}</span>;\n}\n`,
  ).output;
  assert.match(out, /t\("Auto-match: \{0\}", on \? t\("On"\) : t\("Off"\)\)/);
});

test("leaves the key of a t() call alone", () => {
  const result = run(
    `export function Panel() {\n  const t = useT();\n  return <span>{t("Create Shipment")}</span>;\n}\n`,
  );
  assert.equal(result.changed, false, "argument 0 is the catalog key, not a value to wrap");
});

test("wraps a nullish fallback used as a placeholder value", () => {
  const out = run(
    `export function Panel({ card }: any) {\n  const t = useT();\n  return <h2>{t("Cancel {0}?", card?.label ?? "this card")}</h2>;\n}\n`,
  ).output;
  assert.match(out, /card\?\.label \?\? t\("this card"\)/);
});

test("does not wrap a literal in an attribute expression", () => {
  const result = run(
    `export function Panel({ i }: any) {\n  return <Field name={\`lines.\${i}.amount\`} render={() => null} />;\n}\n`,
  );
  assert.equal(result.changed, false, "an attribute value is a field path, not display text");
});

test("does not wrap a literal passed to an unrelated call", () => {
  const result = run(
    `export function Panel() {\n  return <span>{format(value, "long form")}</span>;\n}\n`,
  );
  assert.equal(result.changed, false, "another function's arguments are its own business");
});

test("folds a template literal rendered in jsx child position", () => {
  const out = run(
    `export function Panel({ name }: any) {\n  return <p>{\`Pull this move back from \${name}. The move returns to uncovered.\`}</p>;\n}\n`,
  ).output;
  assert.match(
    out,
    /t\("Pull this move back from \{0\}\. The move returns to uncovered\.", name\)/,
  );
});

test("folds a template literal handed into an existing t() call", () => {
  const out = run(
    `export function Panel({ n }: any) {\n  const t = useT();\n  return <p>{t("Status: {0}", \`\${n} stops remaining\`)}</p>;\n}\n`,
  ).output;
  assert.match(out, /t\("Status: \{0\}", t\("\{0\} stops remaining", n\)\)/);
});

test("numbers every expression of a folded template", () => {
  const out = run(
    `export function Panel({ a, b }: any) {\n  return <p>{\`Moved \${a} of \${b} stops.\`}</p>;\n}\n`,
  ).output;
  assert.match(out, /t\("Moved \{0\} of \{1\} stops\.", a, b\)/);
});

test("leaves a template that is only placeholders alone", () => {
  const result = run(
    `export function Panel({ a, b }: any) {\n  return <p>{\`\${a} \${b}\`}</p>;\n}\n`,
  );
  assert.equal(result.changed, false, "a bare concatenation carries no words to translate");
});

test("leaves a template literal in an attribute alone", () => {
  const result = run(
    `export function Panel({ i }: any) {\n  return <Field name={\`Line \${i} amount\`} />;\n}\n`,
  );
  assert.equal(result.changed, false, "an attribute value is not display text");
});

test("folds a template literal inside a ternary branch", () => {
  const out = run(
    `export function Panel({ many, n }: any) {\n  return <p>{many ? \`\${n} shipments selected\` : "One shipment selected"}</p>;\n}\n`,
  ).output;
  assert.match(out, /many \? t\("\{0\} shipments selected", n\) : t\("One shipment selected"\)/);
});

test("leaves the contents of a style element alone", () => {
  const result = run(
    `export function Spinner() {\n  return <svg><style>{".bar { animation: spin 0.8s linear infinite; }"}</style></svg>;\n}\n`,
  );
  assert.equal(result.changed, false, "a style element's children are CSS, not display text");
});

test("leaves style element text alone", () => {
  const result = run(
    `export function Spinner() {\n  return <svg><style>.bar {'{'} color: red; {'}'}</style></svg>;\n}\n`,
  );
  assert.equal(result.changed, false);
});

test("keeps the space that separates a folded template from its sibling", () => {
  const out = run(
    `export function Row({ n }: any) {\n  return <span>{count}{n > 0 ? \` · \${n} open\` : ""}</span>;\n}\n`,
  ).output;
  assert.match(out, /\` \$\{t\("· \{0\} open", n\)\}\`/);
});

test("does not pad a template that had no surrounding whitespace", () => {
  const out = run(
    `export function Row({ n }: any) {\n  return <span>{n > 0 ? \`\${n} open\` + "!" : ""}</span>;\n}\n`,
  ).output;
  assert.doesNotMatch(out, /\`\$\{t\(/);
});

test("does not invent a space where JSX removed the newline between interpolations", () => {
  const out = run(
    `export function Row({ name, title }: any) {\n  return (\n    <p>\n      by {name}\n      {title ? \`, \${title}\` : ""}\n    </p>\n  );\n}\n`,
  ).output;
  assert.match(out, /t\("by \{0\}\{1\}"/);
});

test("keeps the space where the interpolations sat on one line", () => {
  const out = run(
    `export function Row({ first, last }: any) {\n  return <p>Driver {first} {last} is here</p>;\n}\n`,
  ).output;
  assert.match(out, /t\("Driver \{0\} \{1\} is here"/);
});
