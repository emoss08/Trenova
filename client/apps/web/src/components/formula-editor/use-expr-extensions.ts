import { Compartment, type Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { useEffect, useMemo, useRef, type RefObject } from "react";
import { createExprHover, type HoverVariable } from "./expr-hover";
import { exprLanguageSupport } from "./expr-language";
import { createExprLinter } from "./expr-lint";
import type { KnownIdentifiers } from "./known-identifiers";

/**
 * Builds the editor's extensions once and swaps only the language and
 * completion source when the known identifiers change. Without the
 * compartment every keystroke in a custom-variable name rebuilt the whole
 * extension set for the main editor and every breakdown line, and CodeMirror
 * reconfigured all of them.
 */
export function useExprExtensions(
  known: KnownIdentifiers,
  lint: boolean,
  viewRef: RefObject<EditorView | null>,
  hoverValues?: () => ReadonlyMap<string, HoverVariable>,
): Extension[] {
  const compartment = useRef(new Compartment()).current;
  const initialKnown = useRef(known);
  const knownRef = useRef(known);
  knownRef.current = known;
  const hoverValuesRef = useRef(hoverValues);
  hoverValuesRef.current = hoverValues;

  const extensions = useMemo(() => {
    const list: Extension[] = [
      compartment.of(exprLanguageSupport(initialKnown.current)),
      EditorView.lineWrapping,
      createExprHover(() => hoverValuesRef.current?.() ?? new Map()),
    ];
    if (lint) {
      list.push(createExprLinter(() => knownRef.current));
    }
    return list;
  }, [compartment, lint]);

  useEffect(() => {
    if (known === initialKnown.current) return;
    const view = viewRef.current;
    if (!view) return;
    view.dispatch({ effects: compartment.reconfigure(exprLanguageSupport(known)) });
  }, [known, compartment, viewRef]);

  return extensions;
}
