// tariff-inlays.ts draws inlay hints: each binding's inferred type from `tariff.inlays` where the
// annotation would go (`let miles: Distance = …`), and, after a binding's line, what it came to
// in the last preview from the trace (`= 812 mi`). Both are widgets, so they are never part of
// the text and cannot be selected or copied.

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
import type { TariffTraceEvent } from "@trenova/shared/types/tariff";
import { unlessUnanswered } from "./tariff-doc";
import type { TariffLanguageService } from "./tariff-service";

/** Tells the editor the preview's trace changed, so the value hints are redrawn. */
export const refreshTariffInlays = StateEffect.define<null>();

class InlayWidget extends WidgetType {
  constructor(
    readonly text: string,
    readonly kind: "type" | "value",
  ) {
    super();
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

/** The value hints: each bound name's last value, at the end of its statement's last line. */
export function valueInlays(
  trace: readonly TariffTraceEvent[],
  length: number,
): { position: number; label: string }[] {
  const last = new Map<number, { position: number; label: string }>();
  for (const event of trace) {
    if (event.kind !== "bind" && event.kind !== "assign") continue;
    const position = Math.min(event.range.to, length);
    last.set(event.range.from, { position, label: `${event.name} = ${event.display}` });
  }
  return [...last.values()].sort((a, b) => a.position - b.position);
}

type Hint = { position: number; label: string; kind: "type" | "value"; side: number };

function build(
  service: TariffLanguageService,
  view: EditorView,
  types: readonly TariffInlay[],
): DecorationSet {
  const length = view.state.doc.length;
  const hints: Hint[] = [
    ...types.map((inlay) => ({
      position: Math.min(inlay.position, length),
      label: inlay.label,
      kind: inlay.kind,
      side: 1,
    })),
    ...valueInlays(service.trace(), length).map((inlay) => ({
      position: view.state.doc.lineAt(inlay.position).to,
      label: inlay.label,
      kind: "value" as const,
      side: 2,
    })),
  ].sort((a, b) => a.position - b.position || a.side - b.side);
  const builder = new RangeSetBuilder<Decoration>();
  for (const hint of hints) {
    builder.add(
      hint.position,
      hint.position,
      Decoration.widget({ widget: new InlayWidget(hint.label, hint.kind), side: hint.side }),
    );
  }
  return builder.finish();
}

function typeInlays(service: TariffLanguageService, view: EditorView): TariffInlay[] {
  const from = view.visibleRanges[0]?.from ?? 0;
  const to = view.visibleRanges.at(-1)?.to ?? view.state.doc.length;
  return unlessUnanswered(
    () =>
      tariffInlays(service.rules(), {
        source: view.state.doc.toString(),
        kind: service.kind(),
        context: service.context(),
        from,
        to,
      }),
    [],
  );
}

/** Inlay hints, redrawn when the text, the viewport or the preview's trace changes. */
export function tariffInlayHints(service: TariffLanguageService): Extension {
  return ViewPlugin.fromClass(
    class {
      types: TariffInlay[];
      decorations: DecorationSet;

      constructor(view: EditorView) {
        this.types = typeInlays(service, view);
        this.decorations = build(service, view, this.types);
      }

      update(update: ViewUpdate) {
        const refreshed = update.transactions.some((transaction) =>
          transaction.effects.some((effect) => effect.is(refreshTariffInlays)),
        );
        if (update.docChanged || update.viewportChanged) {
          this.types = typeInlays(service, update.view);
        }
        if (update.docChanged || update.viewportChanged || refreshed) {
          this.decorations = build(service, update.view, this.types);
        }
      }
    },
    { decorations: (value) => value.decorations },
  );
}
