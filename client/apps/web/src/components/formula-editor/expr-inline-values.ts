import { StateEffect, StateField } from "@codemirror/state";
import { Decoration, EditorView, WidgetType, type DecorationSet } from "@codemirror/view";

/*
 * Inline values: what each statement came to on the preview load, drawn at
 * the end of its line the way a notebook shows a cell's result. The studio
 * sets them from the engine's trace after every preview; they are never
 * computed in the browser.
 */

export type InlineValue = {
  /** The offset the value is drawn after: the end of its statement. */
  at: number;
  text: string;
  tone: "ok" | "bad" | "muted";
};

export const setInlineValues = StateEffect.define<readonly InlineValue[]>();

class InlineValueWidget extends WidgetType {
  private readonly text: string;
  private readonly tone: InlineValue["tone"];

  constructor(text: string, tone: InlineValue["tone"]) {
    super();
    this.text = text;
    this.tone = tone;
  }

  eq(other: InlineValueWidget): boolean {
    return other.text === this.text && other.tone === this.tone;
  }

  toDOM(): HTMLElement {
    const span = document.createElement("span");
    // i18n-ignore: CSS class names, not text
    span.className = `cm-expr-inline-value cm-expr-inline-value-${this.tone}`;
    span.textContent = this.text;
    span.setAttribute("aria-hidden", "true");
    return span;
  }

  ignoreEvent(): boolean {
    return true;
  }
}

function build(values: readonly InlineValue[], length: number): DecorationSet {
  const decorations = values
    .filter((value) => value.at >= 0 && value.at <= length)
    .sort((a, b) => a.at - b.at)
    .map((value) =>
      Decoration.widget({ widget: new InlineValueWidget(value.text, value.tone), side: 1 }).range(
        value.at,
      ),
    );
  return Decoration.set(decorations, true);
}

const inlineValuesField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update: (decorations, transaction) => {
    let next = decorations.map(transaction.changes);
    for (const effect of transaction.effects) {
      if (effect.is(setInlineValues)) {
        next = build(effect.value, transaction.state.doc.length);
      }
    }
    if (transaction.docChanged) {
      // An edit makes the values stale until the preview runs again.
      next = Decoration.none;
    }
    return next;
  },
  provide: (field) => EditorView.decorations.from(field),
});

export function inlineValuesExtension() {
  return inlineValuesField;
}
