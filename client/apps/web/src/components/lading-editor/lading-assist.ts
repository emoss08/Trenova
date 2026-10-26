// lading-assist.ts is the Lading editor's help at the cursor, all answered by the EDI module:
// completion (`lading.complete`: X12 paths from the partner's guide, document fields, settings,
// functions, crosswalks, variants, snippets with tab stops), the hover card (`lading.hover`:
// the guide's description, the type, the doc and the last run's value), go to definition
// (`lading.definition`, F12 or Mod-click, into another file of the project), folding from the
// check's outline, and formatting (`lading.format`, Shift-Alt-F).

import {
  autocompletion,
  snippet,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from "@codemirror/autocomplete";
import { foldService } from "@codemirror/language";
import { Prec, type Extension } from "@codemirror/state";
import { EditorView, hoverTooltip, keymap, type Command, type Tooltip } from "@codemirror/view";
import {
  ladingComplete,
  ladingDefinition,
  ladingDocText,
  ladingFormat,
  ladingHover,
  type LadingCompletionItem,
  type LadingHover,
} from "@trenova/shared/edi/lading";
import type { CatalogMessage } from "@trenova/shared/i18n/message";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import { codeMirrorSnippet } from "../formula-editor/tariff-complete";
import { foldAt } from "../formula-editor/tariff-fold";
import { FORMAT_KEY } from "../formula-editor/tariff-format";
import {
  checkedFile,
  ladingServiceFacet,
  unlessRefused,
  type LadingLanguageService,
} from "./lading-service";

/** CodeMirror's icon class for each kind the module completes. */
const COMPLETION_TYPE: Record<string, string> = {
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

function docText(doc: LadingCompletionItem["doc"]): string {
  return ladingDocText(doc, (message: CatalogMessage) => translateMessage(message));
}

function at(service: LadingLanguageService, offset: number) {
  return {
    project: service.project(),
    context: service.context(),
    path: service.path(),
    offset,
  };
}

/** What the module offers at the cursor, as CodeMirror completions. */
export function ladingCompletions(
  service: LadingLanguageService,
  position: number,
): CompletionResult | null {
  const answer = unlessRefused(() => ladingComplete(service.edi(), at(service, position)), null);
  if (!answer || answer.items.length === 0) return null;
  const count = answer.items.length;
  const options: Completion[] = answer.items.map((item, index) => ({
    label: item.label,
    apply: item.snippet ? snippet(codeMirrorSnippet(item.insert)) : item.insert,
    type: COMPLETION_TYPE[item.kind] ?? "text",
    boost: Math.round(((count - index) / count) * 99) - 99,
    ...(item.detail ? { detail: item.detail } : {}),
    ...(item.doc
      ? {
          info: () => {
            const dom = document.createElement("div");
            dom.className = "cm-lading-completion-info";
            dom.textContent = docText(item.doc);
            return dom;
          },
        }
      : {}),
  }));
  return { from: answer.range.from, to: answer.range.to, options, validFor: /^[\w.]*$/ };
}

/** The hover card's element. */
export function renderLadingHover(hover: LadingHover): HTMLElement {
  const { contents } = hover;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-hover";
  if (contents.signature || contents.type) {
    const head = document.createElement("div");
    head.className = "cm-tariff-hover-head";
    if (contents.signature) {
      const signature = document.createElement("code");
      signature.className = "cm-tariff-hover-title";
      signature.textContent = contents.signature;
      head.appendChild(signature);
    }
    if (contents.type && contents.type !== contents.signature) {
      const type = document.createElement("span");
      type.className = "cm-tariff-hover-type";
      type.textContent = contents.type;
      head.appendChild(type);
    }
    dom.appendChild(head);
  }
  const doc = docText(contents.doc ?? undefined);
  if (doc) {
    const body = document.createElement("div");
    body.className = "cm-tariff-hover-body";
    body.textContent = doc;
    dom.appendChild(body);
  }
  if (contents.value) {
    const row = document.createElement("div");
    row.className = "cm-tariff-hover-value";
    row.textContent = translate("In the last run: {0}", contents.value.value);
    dom.appendChild(row);
  }
  return dom;
}

function hoverAt(service: LadingLanguageService, position: number): LadingHover | null {
  return unlessRefused(
    () => ladingHover(service.edi(), { ...at(service, position), values: [...service.values()] }),
    null,
  );
}

/** Goes to the declaration of the name at the cursor; false when there is none. */
export const goToLadingDefinition: Command = (view) => {
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const targets = unlessRefused(
    () => ladingDefinition(service.edi(), at(service, view.state.selection.main.head)),
    [],
  );
  const target = targets[0];
  if (!target) return false;
  service.openLocation(target);
  return true;
};

/** Formats the document in place; false (and the reason) when the file has syntax errors. */
export function formatLadingDocument(
  view: EditorView,
  service: LadingLanguageService,
): { formatted: boolean; reason: string | null } {
  const source = view.state.doc.toString();
  const result = unlessRefused(() => ladingFormat(service.edi(), source), null);
  if (!result) return { formatted: false, reason: null };
  if ("error" in result) return { formatted: false, reason: translateMessage(result.error) };
  if (result.text !== source) {
    view.dispatch({
      changes: { from: 0, to: source.length, insert: result.text },
      userEvent: "format",
    });
  }
  return { formatted: true, reason: null };
}

/** Moves the cursor to a range of this file and shows it. */
export function revealLadingRange(view: EditorView, range: { from: number; to: number }) {
  const to = Math.min(range.to, view.state.doc.length);
  const from = Math.min(range.from, to);
  view.dispatch({
    selection: { anchor: from, head: to },
    effects: EditorView.scrollIntoView(from, { y: "center" }),
  });
  view.focus();
}

export function ladingAssist(
  service: LadingLanguageService,
  onFormat: (reason: string | null) => void,
): Extension {
  const format: Command = (view) => {
    if (view.state.readOnly) return false;
    onFormat(formatLadingDocument(view, service).reason);
    return true;
  };
  return [
    ladingServiceFacet.of(service),
    autocompletion({
      activateOnTyping: true,
      override: [
        (context: CompletionContext) => {
          if (!context.explicit && !context.matchBefore(/[\w.]$/)) return null;
          return ladingCompletions(service, context.pos);
        },
      ],
    }),
    hoverTooltip((_view, position): Tooltip | null => {
      const hover = hoverAt(service, position);
      if (!hover) return null;
      return {
        pos: hover.range.from,
        end: hover.range.to,
        above: true,
        create: () => ({ dom: renderLadingHover(hover) }),
      };
    }),
    foldService.of((_state, lineStart, lineEnd) => {
      const file = checkedFile(service);
      return file ? foldAt(file.folds, lineStart, lineEnd) : null;
    }),
    EditorView.domEventHandlers({
      mousedown(event, view) {
        if (!(event.metaKey || event.ctrlKey)) return false;
        const position = view.posAtCoords({ x: event.clientX, y: event.clientY });
        if (position === null) return false;
        view.dispatch({ selection: { anchor: position } });
        return goToLadingDefinition(view);
      },
    }),
    Prec.high(
      keymap.of([
        { key: "F12", run: goToLadingDefinition },
        { key: FORMAT_KEY, run: format },
      ]),
    ),
  ];
}
