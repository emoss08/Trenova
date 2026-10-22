import { lintGutter } from "@codemirror/lint";
import { Compartment, Prec, type Extension } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import type { FormulaFormatResult } from "@trenova/shared/rules/formula";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { useEffect, useMemo, useState, type RefObject } from "react";
import { FORMAT_KEY, formatEditorDocument } from "./editor-format";
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
  /** Told what formatting from the keyboard did, so the page can say why it was refused. */
  onFormat?: (result: FormulaFormatResult) => void;
};

/**
 * Builds the editor's extensions once and swaps only the highlighter and
 * completion source when the known identifiers change. Without the
 * compartment every keystroke in a custom-variable name rebuilt the whole
 * extension set for the main editor and every breakdown line, and CodeMirror
 * reconfigured all of them. The linter, hover, signature help and the format
 * key read the loaded rules module and the current identifiers through
 * getters for the same reason.
 */
export function useExprExtensions(
  rules: RulesRuntime,
  known: KnownIdentifiers,
  lint: boolean,
  viewRef: RefObject<EditorView | null>,
  options: ExprExtensionOptions = {},
): Extension[] {
  const [compartment] = useState(() => new Compartment());
  const [initialKnown] = useState(known);
  const getRules = useLatestCallback(() => rules);
  const getKnown = useLatestCallback(() => known);
  const getHoverValues = useLatestCallback(
    (): ReadonlyMap<string, HoverVariable> => options.hoverValues?.() ?? new Map(),
  );
  const reportFormat = useLatestCallback((result: FormulaFormatResult) =>
    options.onFormat?.(result),
  );
  const getDeclareInput = useLatestCallback(() => options.declareInput);
  const gutter = options.gutter ?? false;

  const extensions = useMemo(() => {
    const list: Extension[] = [
      compartment.of(exprLanguageSupport(initialKnown)),
      EditorView.lineWrapping,
      createExprHover(getRules, getHoverValues, getKnown),
      createSignatureHelp(getRules, getKnown),
      Prec.high(
        keymap.of([
          {
            key: FORMAT_KEY,
            run: (view) => {
              reportFormat(formatEditorDocument(view, getRules()));
              return true;
            },
          },
        ]),
      ),
    ];
    if (lint) {
      list.push(createExprLinter(getRules, getKnown, getDeclareInput));
      if (gutter) list.push(lintGutter());
    }
    return list;
  }, [
    compartment,
    initialKnown,
    getRules,
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
