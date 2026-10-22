// tariff-inlays.ts draws `tariff.inlays`' hints for what is on screen: each binding's inferred
// type where the annotation would go (`let miles: Distance = …`), parameter names before
// positional arguments, and, given the last preview's trace, what each binding came to at the end
// of its line. All are widgets, so they are never part of the text and cannot be copied.

import { RangeSetBuilder, StateEffect, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  WidgetType,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";
import { tariffInlays, type TariffInlay } from "@trenova/shared/rules/tariff";
import { unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

/** Tells the editor the preview's trace changed, so the value hints are redrawn. */
export const refreshTariffInlays = StateEffect.define<null>();

class InlayWidget extends WidgetType {
  readonly text: string;
  readonly kind: "type" | "value";

  constructor(text: string, kind: "type" | "value") {
    super();
    this.text = text;
    this.kind = kind;
  }

  eq(other: InlayWidget) {
    return other.text === this.text && other.kind === this.kind;
  }

  toDOM() {
    const dom = document.createElement("span");
    dom.className = `cm-tariff-inlay cm-tariff-inlay-${this.kind}`;
    dom.textContent = this.text;
    dom.setAttribute("aria-hidden", "true");
    return dom;
  }

  ignoreEvent() {
    return true;
  }
}

function build(view: EditorView, hints: readonly TariffInlay[]): DecorationSet {
  const length = view.state.doc.length;
  const builder = new RangeSetBuilder<Decoration>();
  const sorted = [...hints].sort((a, b) => a.offset - b.offset);
  for (const hint of sorted) {
    const position = Math.min(hint.offset, length);
    const kind = hint.kind === "value" ? "value" : "type";
    builder.add(
      position,
      position,
      Decoration.widget({
        widget: new InlayWidget(hint.label, kind),
        side: hint.kind === "parameter" ? -1 : 1,
      }),
    );
  }
  return builder.finish();
}

function visibleInlays(service: TariffLanguageService, view: EditorView): TariffInlay[] {
  const from = view.visibleRanges[0]?.from ?? 0;
  const to = view.visibleRanges.at(-1)?.to ?? view.state.doc.length;
  return unlessUnanswered(
    () =>
      tariffInlays(service.rules(), {
        source: view.state.doc.toString(),
        context: service.context(),
        from,
        to,
        values: service.trace(),
      }),
    [],
  );
}

/** Inlay hints, redrawn when the text, the viewport or the preview's trace changes. */
export function tariffInlayHints(service: TariffLanguageService): Extension {
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;

      constructor(view: EditorView) {
        this.decorations = build(view, visibleInlays(service, view));
      }

      update(update: ViewUpdate) {
        const refreshed = update.transactions.some((transaction) =>
          transaction.effects.some((effect) => effect.is(refreshTariffInlays)),
        );
        if (update.docChanged || update.viewportChanged || refreshed) {
          this.decorations = build(update.view, visibleInlays(service, update.view));
        }
      }
    },
    { decorations: (value) => value.decorations },
  );
}
