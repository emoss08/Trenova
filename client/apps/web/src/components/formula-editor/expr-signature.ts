import { StateEffect, StateField, type EditorState, type Extension } from "@codemirror/state";
import { EditorView, showTooltip, type Tooltip } from "@codemirror/view";
import { nodePath, parseExpression, memberPath, type Node } from "./expr-parser";
import type { FunctionDoc, KnownIdentifiers } from "./known-identifiers";

/*
 * Signature help: while the cursor sits inside a call's parentheses, the
 * function's signature floats above it with the argument being typed in
 * bold, the way an IDE does it.
 */

export type SignatureHelp = {
  fn: FunctionDoc;
  /** Which argument the cursor is in, from 0. */
  active: number;
  /** Where the tooltip anchors: the opening parenthesis. */
  at: number;
};

/** The call the cursor is inside, innermost first, and which argument it is on. */
export function signatureAt(
  expression: string,
  position: number,
  known: KnownIdentifiers,
): SignatureHelp | null {
  const program = parseExpression(expression, known.operatorNames);
  const path = nodePath(program, position);
  for (let index = path.length - 1; index >= 0; index -= 1) {
    const node = path[index];
    if (node.kind !== "call") continue;
    if (
      position <= node.parens.from ||
      position >
        node.parens.to -
          (node.parens.to > node.parens.from && expression[node.parens.to - 1] === ")" ? 1 : 0)
    ) {
      continue;
    }
    const name = memberPath(node.callee);
    const fn = name ? known.functions.find((item) => item.name === name) : undefined;
    if (!fn || fn.operator) continue;
    return { fn, active: activeArgument(node, position, expression), at: node.parens.from };
  }
  return null;
}

function activeArgument(
  call: Node & { kind: "call" },
  position: number,
  expression: string,
): number {
  let active = 0;
  for (let index = 0; index < call.args.length; index += 1) {
    const arg = call.args[index];
    if (position > arg.to) {
      const between = expression.slice(arg.to, Math.min(position, call.parens.to));
      if (between.includes(",")) active = index + 1;
      else active = index;
    } else {
      return index;
    }
  }
  return Math.max(active, 0);
}

/** The signature split into its parameters, so one can be drawn bold. */
export function signatureParts(signature: string): { name: string; params: string[] } {
  const open = signature.indexOf("(");
  const close = signature.lastIndexOf(")");
  if (open === -1 || close <= open) return { name: signature, params: [] };
  const params = signature
    .slice(open + 1, close)
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "");
  return { name: signature.slice(0, open), params };
}

function render(help: SignatureHelp): HTMLElement {
  const dom = document.createElement("div");
  dom.className = "cm-expr-signature";
  const { name, params } = signatureParts(help.fn.signature);
  const code = document.createElement("code");
  code.className = "cm-expr-signature-code";
  code.appendChild(document.createTextNode(`${name}(`));
  const variadicIndex = params.findIndex((param) => param.startsWith("..."));
  params.forEach((param, index) => {
    const active =
      index === help.active ||
      (variadicIndex !== -1 && help.active >= variadicIndex && index === variadicIndex);
    const span = document.createElement("span");
    span.className = active
      ? "cm-expr-signature-param cm-expr-signature-active"
      : "cm-expr-signature-param";
    span.textContent = param;
    code.appendChild(span);
    if (index < params.length - 1) code.appendChild(document.createTextNode(", "));
  });
  code.appendChild(document.createTextNode(")"));
  dom.appendChild(code);
  if (help.fn.description) {
    const body = document.createElement("div");
    body.className = "cm-expr-signature-body";
    body.textContent = help.fn.description;
    dom.appendChild(body);
  }
  return dom;
}

function tooltipFor(state: EditorState, getKnown: () => KnownIdentifiers): Tooltip | null {
  const selection = state.selection.main;
  if (!selection.empty) return null;
  const help = signatureAt(state.doc.toString(), selection.head, getKnown());
  if (!help) return null;
  return {
    pos: help.at,
    above: true,
    strictSide: true,
    arrow: false,
    create: () => ({ dom: render(help) }),
  };
}

/** The editor gained or lost focus; a tooltip belongs only to a focused editor. */
export const setEditorFocused = StateEffect.define<boolean>();

/** Carries focus changes into the state, for every tooltip that must leave with the focus. */
export const editorFocusTracking: Extension = EditorView.focusChangeEffect.of((_state, focusing) =>
  setEditorFocused.of(focusing),
);

/** Whether a transaction says the editor lost focus. */
export function lostFocus(transaction: { effects: readonly StateEffect<unknown>[] }): boolean {
  return transaction.effects.some((effect) => effect.is(setEditorFocused) && !effect.value);
}

type SignatureState = { focused: boolean; tooltip: Tooltip | null };

/**
 * Signature help, kept in step with the cursor and the document, and shown
 * only while the editor has focus: a dialog opening over the editor takes
 * the focus with it, and the help must not stay on screen above the dialog.
 */
export function createSignatureHelp(getKnown: () => KnownIdentifiers): Extension {
  const field = StateField.define<SignatureState>({
    create: () => ({ focused: false, tooltip: null }),
    update: (value, transaction) => {
      let focused = value.focused;
      for (const effect of transaction.effects) {
        if (effect.is(setEditorFocused)) focused = effect.value;
      }
      if (!focused) {
        return value.tooltip === null && focused === value.focused
          ? value
          : { focused, tooltip: null };
      }
      if (!transaction.docChanged && !transaction.selection && focused === value.focused) {
        return value;
      }
      return { focused, tooltip: tooltipFor(transaction.state, getKnown) };
    },
    provide: (field) => showTooltip.from(field, (value) => value.tooltip),
  });
  return [field, editorFocusTracking];
}
