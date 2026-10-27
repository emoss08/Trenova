// lading-complete.ts is completion from `lading.complete`, asked about the text as the editor
// holds it (the studio learns of an edit after it): X12 paths from the partner's guide, document
// fields, settings, functions and methods, crosswalks, variants, events, the codes an element
// takes inside the quotes it is compared with, and snippets with tab stops. The list keeps the
// module's order, and each item shows its signature and its doc beside it.

import {
  autocompletion,
  snippet,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import type { Extension } from "@codemirror/state";
import { ladingComplete, type LadingCompletionItem } from "@trenova/shared/edi/lading";
import { codeMirrorSnippet } from "../formula-editor/tariff-complete";
import { appendLadingCode, appendLadingDoc } from "./lading-doc";
import { ladingRequest, unlessRefused, type LadingLanguageService } from "./lading-service";

/** CodeMirror's icon class for each kind the module completes. */
export const LADING_COMPLETION_TYPE: Record<string, string> = {
  field: "property",
  variable: "variable",
  function: "function",
  method: "method",
  segment: "class",
  element: "property",
  loop: "namespace",
  keyword: "keyword",
  type: "type",
  variant: "enum",
  crosswalk: "function",
  constant: "constant",
  setting: "variable",
  event: "enum",
  map: "namespace",
  rules: "namespace",
  lookup: "function",
  file: "namespace",
  snippet: "text",
};

/** The kinds whose detail is Lading (a signature or a type), drawn in the syntax colours. */
const CODE_DETAIL = new Set(["function", "method", "variable", "field", "setting", "crosswalk"]);

function info(service: LadingLanguageService, item: LadingCompletionItem): Completion["info"] {
  if (!item.doc && !(item.detail && CODE_DETAIL.has(item.kind))) return undefined;
  return () => {
    const dom = document.createElement("div");
    dom.className = "cm-tariff-completion-info";
    if (item.detail && CODE_DETAIL.has(item.kind)) {
      const code = document.createElement("code");
      appendLadingCode(code, item.detail, service.edi());
      dom.appendChild(code);
    }
    const body = document.createElement("div");
    if (appendLadingDoc(body, item.doc)) dom.appendChild(body);
    return dom;
  };
}

/** What the module offers at `position` of `source`, as CodeMirror completions. */
export function ladingCompletions(
  service: LadingLanguageService,
  source: string,
  position: number,
): CompletionResult | null {
  const answer = unlessRefused(
    () => ladingComplete(service.edi(), ladingRequest(service, source, position)),
    null,
  );
  if (!answer || answer.items.length === 0) return null;
  const count = answer.items.length;
  const options: Completion[] = answer.items.map((item, index) => {
    const detail = info(service, item);
    return {
      label: item.label,
      apply: item.snippet ? snippet(codeMirrorSnippet(item.insert)) : item.insert,
      type: LADING_COMPLETION_TYPE[item.kind] ?? "text",
      boost: Math.round(((count - index) / count) * 99) - 99,
      ...(item.detail ? { detail: item.detail } : {}),
      ...(detail ? { info: detail } : {}),
    };
  });
  const from = Math.min(answer.range.from, position);
  const quoted = source.charAt(from - 1) === '"';
  return {
    from,
    to: Math.max(answer.range.to, position),
    options,
    // Typing more of the same word filters this list; a `.` or a quote asks the module again,
    // since what follows it is another list.
    validFor: quoted ? /^[^"\\]*$/ : /^\w*$/,
  };
}

/** Whether typing just now should open the list: a word, `.`, or an opening quote. */
export function opensLadingCompletion(before: string): boolean {
  return /[\w."]$/.test(before);
}

/** Completion while typing a word, after `.` or inside a compared text, and on Ctrl-Space. */
export function ladingAutocomplete(service: LadingLanguageService): Extension {
  return autocompletion({
    activateOnTyping: true,
    icons: true,
    maxRenderedOptions: 200,
    override: [
      (context: CompletionContext) => {
        if (!context.explicit) {
          const line = context.state.doc.lineAt(context.pos);
          if (!opensLadingCompletion(line.text.slice(0, context.pos - line.from))) return null;
        }
        return ladingCompletions(service, context.state.doc.toString(), context.pos);
      },
    ],
  });
}
