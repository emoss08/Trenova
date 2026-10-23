// tariff-references.ts is everything the editor does with `tariff.references`: once the cursor
// has rested on a name, every use of it in the file is marked (a read in one tint, a write or the
// declaration in another, so where a value changes stands out), and Find references (Shift-F12,
// or the editor's menu) hands the whole list, with its uses in libraries, to the page to show.

import { StateEffect, StateField, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  keymap,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffReferences, type TariffReference } from "@trenova/shared/rules/tariff";
import { unlessUnanswered } from "./tariff-doc";
import { showEditorNotice } from "./tariff-notice";
import { tariffServiceFacet, type TariffLanguageService } from "./tariff-service";

/** How long the cursor rests on a name before its uses are marked. */
export const OCCURRENCE_DELAY_MS = 300;

const setOccurrences = StateEffect.define<DecorationSet>();

const readMark = Decoration.mark({ class: "cm-tariff-occurrence" });
const writeMark = Decoration.mark({ class: "cm-tariff-occurrence cm-tariff-occurrence-write" });

const occurrenceField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(decorations, transaction) {
    for (const effect of transaction.effects) if (effect.is(setOccurrences)) return effect.value;
    // An edit makes the marks stale; they come back once the cursor rests again.
    return transaction.docChanged ? Decoration.none : decorations;
  },
  provide: (field) => EditorView.decorations.from(field),
});

/** The uses of the name at `offset` in this file, as marks; none off a name. */
export function occurrenceMarks(
  references: readonly TariffReference[],
  length: number,
): DecorationSet {
  const marks = references
    .filter((reference) => reference.module === null)
    .filter((reference) => reference.range.from < reference.range.to)
    .filter((reference) => reference.range.to <= length)
    .map((reference) =>
      (reference.kind === "read" ? readMark : writeMark).range(
        reference.range.from,
        reference.range.to,
      ),
    );
  return Decoration.set(marks, true);
}

function referencesAt(
  service: TariffLanguageService,
  view: EditorView,
  offset: number,
): TariffReference[] {
  return unlessUnanswered(
    () =>
      tariffReferences(service.rules(), {
        source: view.state.doc.toString(),
        offset,
        context: service.context(),
        kind: service.kind(),
      }),
    [],
  );
}

function occurrencePlugin(service: TariffLanguageService) {
  return ViewPlugin.fromClass(
    class {
      timer: number | null = null;
      readonly view: EditorView;

      constructor(view: EditorView) {
        this.view = view;
        this.schedule();
      }

      update(update: ViewUpdate) {
        if (update.docChanged || update.selectionSet) this.schedule();
      }

      schedule() {
        if (this.timer !== null) window.clearTimeout(this.timer);
        this.timer = window.setTimeout(() => {
          this.timer = null;
          this.mark();
        }, OCCURRENCE_DELAY_MS);
      }

      mark() {
        const { view } = this;
        const selection = view.state.selection.main;
        const current = view.state.field(occurrenceField);
        // A selection that spans more than one word is not "a name at the cursor".
        const word = view.state.wordAt(selection.head);
        const onName =
          selection.empty ||
          (word !== null && selection.from >= word.from && selection.to <= word.to);
        const next = onName
          ? occurrenceMarks(referencesAt(service, view, selection.head), view.state.doc.length)
          : Decoration.none;
        if (next.size === 0 && current.size === 0) return;
        view.dispatch({ effects: setOccurrences.of(next) });
      }

      destroy() {
        if (this.timer !== null) window.clearTimeout(this.timer);
      }
    },
  );
}

/** The text of the name a reference list is about: the use under the cursor, or the first. */
function nameOf(view: EditorView, references: readonly TariffReference[], offset: number): string {
  const local = references.filter((reference) => reference.module === null);
  const here =
    local.find((reference) => reference.range.from <= offset && offset <= reference.range.to) ??
    local[0];
  if (here) return view.state.sliceDoc(here.range.from, here.range.to);
  const word = view.state.wordAt(offset);
  return word ? view.state.sliceDoc(word.from, word.to) : "";
}

/**
 * Find references: every use of the name at the cursor, handed to the page to list. Off a name it
 * says so at the cursor and does nothing else.
 */
export function findReferences(view: EditorView): boolean {
  const service = view.state.facet(tariffServiceFacet);
  if (!service) return false;
  const offset = view.state.selection.main.head;
  const references = referencesAt(service, view, offset);
  if (references.length === 0) {
    showEditorNotice(view, translate("Put the cursor on a name to find its references"));
    return true;
  }
  service.showReferences({ name: nameOf(view, references, offset), offset, references });
  return true;
}

/** The marks at the cursor and the Find references key. */
export function tariffReferenceMarks(service: TariffLanguageService): Extension {
  return [
    tariffServiceFacet.of(service),
    occurrenceField,
    occurrencePlugin(service),
    keymap.of([{ key: "Shift-F12", run: findReferences }]),
  ];
}
