// tariff-complete.ts is completion from `tariff.complete`: the module knows what may be written
// at the cursor (names in scope, fields of a record, methods of a type, units after a number,
// enum variants, table codes and keys, accessorial codes, std modules, snippets), so the editor
// lists what it answers, in its order, and replaces the range it names. A snippet's fields are
// tab stops.

import {
  autocompletion,
  snippet,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import type { Extension } from "@codemirror/state";
import { tariffComplete, type TariffCompletionKind } from "@trenova/shared/rules/tariff";
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
};

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
      }),
    null,
  );
  if (!answer || answer.items.length === 0) return null;
  const count = answer.items.length;
  const options: Completion[] = answer.items.map((item, index) => ({
    label: item.label,
    apply: item.snippet ? snippet(item.insert) : item.insert,
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
