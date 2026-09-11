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

function isMeaningful(child) {
  return !(child.type === "JSXText" && child.value.trim() === "");
}

function isFoldable(child) {
  return child.type === "JSXText" || child.type === "JSXExpressionContainer";
}

/**
 * foldableRuns returns each maximal run of text-and-interpolation children that forms a
 * sentence worth folding: it must contain real text and at least one real interpolation.
 * A run of text alone is an ordinary JSXText and is handled as one; a run of
 * interpolations alone carries nothing to translate.
 */
export function foldableRuns(children) {
  const meaningful = children.filter(isMeaningful);
  const runs = [];

  let current = [];
  const flush = () => {
    if (current.length > 1) runs.push(current);
    current = [];
  };

  for (const child of meaningful) {
    if (isFoldable(child)) {
      current.push(child);
      continue;
    }
    flush();
  }
  flush();

  const folded = [];
  for (const run of runs) {
    const built = buildMessage(run);
    if (built !== null) folded.push(built);
  }
  return folded;
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
    message += `{${index}}`;
    index += 1;
    expressions.push(expr);
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
