import { lintGutter } from "@codemirror/lint";
import { Compartment, Prec, type Extension } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useEffect, useMemo, useState, type RefObject } from "react";
import { formatEditorDocument, type FormatResult } from "./expr-format";
import { createExprHover, type HoverVariable } from "./expr-hover";
import { exprLanguageSupport } from "./expr-language";
import { createExprLinter, type DeclareInput } from "./expr-lint";
import { createSignatureHelp } from "./expr-signature";
import type { KnownIdentifiers } from "./known-identifiers";

/** The key that formats the document, as VS Code binds it. */
export const FORMAT_KEY = "Shift-Alt-f";

export type ExprExtensionOptions = {
  /** What the last preview resolved each name to, for the hover card. */
  hoverValues?: () => ReadonlyMap<string, HoverVariable>;
  /** Declares an unknown name as one of the template's inputs, from a quick fix. */
  declareInput?: DeclareInput;
  /** Draws the gutter's markers; off for a one-line editor. */
  gutter?: boolean;
  /** Told what formatting from the keyboard did, so the page can say why it was refused. */
  onFormat?: (result: FormatResult) => void;
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
  const [compartment] = useState(() => new Compartment());
  const [initialKnown] = useState(known);
  const getKnown = useLatestCallback(() => known);
  const getHoverValues = useLatestCallback(
    (): ReadonlyMap<string, HoverVariable> => resolved.hoverValues?.() ?? new Map(),
  );
  const reportFormat = useLatestCallback((result: FormatResult) => resolved.onFormat?.(result));
  const getDeclareInput = useLatestCallback(() => resolved.declareInput);
  const gutter = resolved.gutter ?? false;

  const extensions = useMemo(() => {
    const list: Extension[] = [
      compartment.of(exprLanguageSupport(initialKnown)),
      EditorView.lineWrapping,
      createExprHover(getHoverValues, getKnown),
      createSignatureHelp(getKnown),
      Prec.high(
        keymap.of([
          {
            key: FORMAT_KEY,
            run: (view) => {
              const result = formatEditorDocument(view, {
                operators: getKnown().operatorNames,
              });
              reportFormat(result);
              return true;
            },
          },
        ]),
      ),
    ];
    if (lint) {
      list.push(createExprLinter(getKnown, getDeclareInput));
      if (gutter) list.push(lintGutter());
    }
    return list;
  }, [
    compartment,
    initialKnown,
    getHoverValues,
    getKnown,
    reportFormat,
    getDeclareInput,
    lint,
    gutter,
  ]);

  useEffect(() => {
    if (known === initialKnown) return;
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({ effects: compartment.reconfigure(exprLanguageSupport(known)) });
  }, [known, initialKnown, compartment, viewRef]);

  return extensions;
}
