// fold.mjs turns a JSX sentence broken up by interpolations into one message with numbered
// placeholders. Both the extractor and the codemod use it, so the key written into the
// catalog and the key wrapped in the source are produced by the same code and cannot drift.
//
//   <p>Delete "{name}"? This cannot be undone.</p>
//     -> 'Delete "{0}"? This cannot be undone.'
//
// Folding matters because the halves are untranslatable on their own: Spanish and Chinese
// order that sentence differently, and half a clause gives a translator nothing to work
// with. It is done over *contiguous runs* rather than whole elements, so an unrelated
// sibling does not defeat it:
//
//   <a><Logo /> Continue with {provider.name}</a>
//     -> 'Continue with {0}', with <Logo /> left exactly where it was.

function isWhitespaceText(child) {
  return child.type === "JSXText" && child.value.trim() === "";
}

function isFoldable(child) {
  return child.type === "JSXText" || child.type === "JSXExpressionContainer";
}

/**
 * foldableRuns returns each maximal run of text-and-interpolation children that forms a
 * sentence worth folding: it must contain real text and at least one real interpolation.
 * A run of text alone is an ordinary JSXText and is handled as one; a run of
 * interpolations alone carries nothing to translate.
 *
 * Whitespace-only text INSIDE a run is kept. It is not layout — it is the space in
 * `{years} {unit} on {date}`, and dropping it silently renders "2years on Mar 3".
 * Only the whitespace at the run's edges is separated out, to be re-emitted around the
 * call rather than absorbed into the message.
 */
export function foldableRuns(children) {
  const runs = [];
  let current = [];

  const flush = () => {
    if (current.length > 0) runs.push(current);
    current = [];
  };

  for (const child of children) {
    if (isFoldable(child)) {
      current.push(child);
      continue;
    }
    flush();
  }
  flush();

  const folded = [];
  for (const run of runs) {
    const built = buildMessage(trimRun(run));
    if (built !== null) folded.push(built);
  }
  return folded;
}

// trimRun drops whitespace-only nodes from each end of a run so the message itself does not
// start or end with layout, while the interior keeps every space it had.
function trimRun(run) {
  let start = 0;
  let end = run.length;
  while (start < end && isWhitespaceText(run[start])) start += 1;
  while (end > start && isWhitespaceText(run[end - 1])) end -= 1;
  return run.slice(start, end);
}

function containsJSX(node) {
  if (node === null || typeof node !== "object") return false;
  if (Array.isArray(node)) return node.some(containsJSX);
  if (typeof node.type !== "string") return false;
  if (node.type === "JSXElement" || node.type === "JSXFragment") return true;

  for (const key of Object.keys(node)) {
    if (key === "loc" || key === "leadingComments" || key === "trailingComments") continue;
    if (containsJSX(node[key])) return true;
  }
  return false;
}

function buildMessage(run) {
  let index = 0;
  let message = "";
  const expressions = [];

  for (const child of run) {
    if (child.type === "JSXText") {
      message += child.value;
      continue;
    }
    const expr = child.expression;
    if (expr.type === "JSXEmptyExpression") continue;
    // A container holding only a string literal is not an interpolation; it is text that
    // happened to be written in braces, so it folds into the message as literal text.
    if (expr.type === "StringLiteral") {
      message += expr.value;
      continue;
    }
    if (containsJSX(expr)) return null;

    message += `{${index}}`;
    index += 1;
    // The container is kept, not the expression: Babel excludes wrapping parentheses from a
    // node's range, so slicing `a && (b)` by the expression yields the unbalanced `a && (b`.
    // The container always spans a complete `{ ... }`.
    expressions.push(child);
  }

  const normalized = message.trim().replace(/\s+/g, " ");
  if (index === 0) return null;
  if (normalized === "") return null;
  // Placeholders alone ("{0} {1}") carry no words to translate.
  if (!/\p{L}/u.test(normalized.replace(/\{\d+\}/g, ""))) return null;

  // The message is trimmed, so the whitespace that separated this run from a neighbouring
  // element has to be handed back to the caller and re-emitted outside the call. Without it
  // `<Logo /> Continue with {x}` loses the space between the icon and the words.
  const first = run[0];
  const last = run[run.length - 1];
  const leading = first.type === "JSXText" ? /^\s*/.exec(first.value)[0] : "";
  const trailing = last.type === "JSXText" ? /\s*$/.exec(last.value)[0] : "";

  return { nodes: run, message: normalized, expressions, leading, trailing };
}
