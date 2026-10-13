import { lintGutter } from "@codemirror/lint";
import { Compartment, type Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { useEffect, useMemo, useRef, type RefObject } from "react";
import { createExprHover, type HoverVariable } from "./expr-hover";
import { exprLanguageSupport } from "./expr-language";
import { createExprLinter, type DeclareInput } from "./expr-lint";
import { createSignatureHelp } from "./expr-signature";
import type { KnownIdentifiers } from "./known-identifiers";

export type ExprExtensionOptions = {
  /** What the last preview resolved each name to, for the hover card. */
  hoverValues?: () => ReadonlyMap<string, HoverVariable>;
  /** Declares an unknown name as one of the template's inputs, from a quick fix. */
  declareInput?: DeclareInput;
  /** Draws the gutter's markers; off for a one-line editor. */
  gutter?: boolean;
};

/**
 * Builds the editor's extensions once and swaps only the language and
 * completion source when the known identifiers change. Without the
 * compartment every keystroke in a custom-variable name rebuilt the whole
 * extension set for the main editor and every breakdown line, and CodeMirror
 * reconfigured all of them. The linter, hover and signature help read the
 * current identifiers through a ref for the same reason.
 */
export function useExprExtensions(
  known: KnownIdentifiers,
  lint: boolean,
  viewRef: RefObject<EditorView | null>,
  options: ExprExtensionOptions | (() => ReadonlyMap<string, HoverVariable>) = {},
): Extension[] {
  const resolved: ExprExtensionOptions =
    typeof options === "function" ? { hoverValues: options } : options;
  const compartment = useRef(new Compartment()).current;
  const initialKnown = useRef(known);
  const knownRef = useRef(known);
  knownRef.current = known;
  const optionsRef = useRef(resolved);
  optionsRef.current = resolved;
  const gutter = resolved.gutter ?? false;

  const extensions = useMemo(() => {
    const getKnown = () => knownRef.current;
    const list: Extension[] = [
      compartment.of(exprLanguageSupport(initialKnown.current)),
      EditorView.lineWrapping,
      createExprHover(() => optionsRef.current.hoverValues?.() ?? new Map(), getKnown),
      createSignatureHelp(getKnown),
    ];
    if (lint) {
      list.push(createExprLinter(getKnown, () => optionsRef.current.declareInput));
      if (gutter) list.push(lintGutter());
    }
    return list;
  }, [compartment, lint, gutter]);

  useEffect(() => {
    if (known === initialKnown.current) return;
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({ effects: compartment.reconfigure(exprLanguageSupport(known)) });
  }, [known, compartment, viewRef]);

  return extensions;
}
