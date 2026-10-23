// tariff-complete.ts is completion from `tariff.complete`: the module knows what may be written
// at the cursor (names in scope, fields of a record, methods of a type, units after a number,
// enum variants, table codes and keys, accessorial codes, std modules, snippets), so the editor
// lists what it answers, in its order, and replaces the range it names. A snippet's fields are
// tab stops.

import {
  autocompletion,
  insertCompletionText,
  pickedCompletion,
  snippet,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import type { Extension } from "@codemirror/state";
import {
  tariffComplete,
  type TariffCompletion,
  type TariffCompletionKind,
} from "@trenova/shared/rules/tariff";
import { appendTariffDoc, unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

/** CodeMirror's icon class for each kind the module completes. */
const COMPLETION_TYPE: Record<TariffCompletionKind, string> = {
  keyword: "keyword",
  name: "variable",
  input: "variable",
  function: "function",
  method: "method",
  property: "property",
  field: "property",
  type: "type",
  enum: "enum",
  variant: "enum",
  unit: "text",
  module: "namespace",
  table: "class",
  code: "constant",
  snippet: "text",
  value: "constant",
};

/**
 * The module writes snippets as LSP does (`$1`, `${1:name}`, `$0` for the final cursor, `\$`,
 * `\}` and `\\` for those characters); CodeMirror reads only braced fields, so a bare `$0`
 * would land as text with the cursor after the call. Every field becomes `${n}` or
 * `${n:name}`, and an escaped character becomes itself.
 */
export function codeMirrorSnippet(lsp: string): string {
  let out = "";
  let index = 0;
  while (index < lsp.length) {
    const char = lsp.charAt(index);
    const next = lsp.charAt(index + 1);
    if (char === "\\" && next !== "" && "$}\\".includes(next)) {
      out += next;
      index += 2;
      continue;
    }
    if (char === "$") {
      const bare = /^\$(\d+)/.exec(lsp.slice(index));
      if (bare) {
        out += `\${${bare[1]}}`;
        index += bare[0].length;
        continue;
      }
      const braced = /^\$\{(\d+)(?::((?:\\.|[^\\}])*))?\}/.exec(lsp.slice(index));
      if (braced) {
        const name = (braced[2] ?? "").replace(/\\([$}\\])/g, "$1");
        out += name ? `\${${braced[1]}:${name}}` : `\${${braced[1]}}`;
        index += braced[0].length;
        continue;
      }
    }
    out += char;
    index += 1;
  }
  return out;
}

/**
 * What a pick writes: the item's text, or its snippet with tab stops. An item with its own
 * range (`.` rewritten to `?.` on an optional) replaces that range instead of the list's.
 */
function applyItem(item: TariffCompletion): Completion["apply"] {
  const range = item.range;
  if (item.snippet) {
    const fill = snippet(codeMirrorSnippet(item.insert));
    return range ? (view, completion) => fill(view, completion, range.from, range.to) : fill;
  }
  if (!range) return item.insert;
  return (view, completion) =>
    view.dispatch({
      ...insertCompletionText(view.state, item.insert, range.from, range.to),
      annotations: pickedCompletion.of(completion),
    });
}

/** What the module offers at the cursor, as CodeMirror completions. */
export function tariffCompletions(
  service: TariffLanguageService,
  source: string,
  position: number,
): CompletionResult | null {
  const answer = unlessUnanswered(
    () =>
      tariffComplete(service.rules(), {
        source,
        offset: position,
        context: service.context(),
        kind: service.kind(),
      }),
    null,
  );
  if (!answer || answer.items.length === 0) return null;
  const count = answer.items.length;
  const options: Completion[] = answer.items.map((item, index) => ({
    label: item.label,
    apply: applyItem(item),
    type: COMPLETION_TYPE[item.kind],
    boost: Math.round(((count - index) / count) * 99) - 99,
    ...(item.detail ? { detail: item.detail } : {}),
    ...(item.doc
      ? {
          info: () => {
            const dom = document.createElement("div");
            dom.className = "cm-tariff-completion-info";
            appendTariffDoc(dom, item.doc);
            return dom;
          },
        }
      : {}),
  }));
  return { from: answer.range.from, to: answer.range.to, options, validFor: /^[\w$.]*$/ };
}

/** Completion while typing a word or after `.`, and on Ctrl-Space anywhere. */
export function tariffAutocomplete(service: TariffLanguageService): Extension {
  return autocompletion({
    activateOnTyping: true,
    override: [
      (context: CompletionContext) => {
        if (!context.explicit && !context.matchBefore(/[\w.$]$/)) return null;
        return tariffCompletions(service, context.state.doc.toString(), context.pos);
      },
    ],
  });
}
