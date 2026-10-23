// tariff-lint.ts draws `tariff.check`'s findings: the underline, the translated message, and
// each fix as an action that applies the module's own source edits. A finding inside an
// imported library is shown on the `use` line that brings that library in, since its range is
// in the library's source; a finding about a contract input value has no place in the source
// and is left to the form that holds the value. The keyboard moves between findings and opens
// the ones on the cursor's line in a card with their fixes.

import { forEachDiagnostic, linter, type Action, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { translateMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import type { TariffDiagnostic, TariffFix } from "@trenova/shared/types/tariff";
import { cursorTooltip, openCursorTooltip } from "./cursor-tooltip";
import { checkService, type TariffLanguageService } from "./tariff-service";

/** Applies a fix's edits, which the module guarantees do not overlap. */
export function applyTariffFix(view: EditorView, fix: Pick<TariffFix, "changes">) {
  view.dispatch({
    changes: fix.changes.map((change) => ({
      from: change.range.from,
      to: change.range.to,
      insert: change.insert,
    })),
    userEvent: "fix",
  });
  view.focus();
}

/** Where a library's finding is shown: the `use` of that module, or the first line. */
function importRange(source: string, module: string): { from: number; to: number } {
  const at = source.indexOf(module);
  if (at >= 0) return { from: at, to: at + module.length };
  const end = source.indexOf("\n");
  return { from: 0, to: end === -1 ? source.length : end };
}

/** A finding as CodeMirror draws it, or null for one with no place in this source. */
export function toTariffLint(diagnostic: TariffDiagnostic, source: string): Diagnostic | null {
  const message = translateMessage(diagnostic.message);
  if (diagnostic.module) {
    const range = importRange(source, diagnostic.module);
    return {
      ...range,
      severity: diagnostic.severity,
      source: diagnostic.code,
      message: translate("In {0}: {1}", diagnostic.module, message),
    };
  }
  if (!diagnostic.range) return null;
  const actions: Action[] = diagnostic.fixes.map((fix) => ({
    name: translateMessage(fix.label),
    apply: (view) => applyTariffFix(view, fix),
  }));
  const length = source.length;
  const from = Math.min(diagnostic.range.from, length);
  return {
    from,
    to: Math.min(Math.max(diagnostic.range.to, from), length),
    severity: diagnostic.severity,
    source: diagnostic.code,
    message,
    ...(actions.length > 0 ? { actions } : {}),
  };
}

/**
 * The linter: the module checks the document 250 ms after the last edit. An empty one is not
 * underlined: there is nothing wrong with it yet.
 */
export function tariffLinter(service: TariffLanguageService): Extension {
  return [
    cursorTooltip,
    linter(
      (view): Diagnostic[] => {
        const source = view.state.doc.toString();
        if (source.trim() === "") return [];
        return checkService(service, source)
          .diagnostics.map((diagnostic) => toTariffLint(diagnostic, source))
          .filter((diagnostic): diagnostic is Diagnostic => diagnostic !== null);
      },
      { delay: 250 },
    ),
  ];
}

type Placed = { diagnostic: Diagnostic; from: number; to: number };

function placedDiagnostics(view: EditorView): Placed[] {
  const placed: Placed[] = [];
  forEachDiagnostic(view.state, (diagnostic, from, to) => placed.push({ diagnostic, from, to }));
  return placed.sort((a, b) => a.from - b.from);
}

/**
 * Moves the cursor to the next (or previous) finding, wrapping at the end of the file, and only
 * to findings of `severity` when given (`]d`, `]e` and `]w` in Vim). False when there is none.
 */
export function jumpToDiagnostic(
  view: EditorView,
  direction: 1 | -1,
  severity?: Diagnostic["severity"],
): boolean {
  const found = placedDiagnostics(view).filter(
    (entry) => !severity || entry.diagnostic.severity === severity,
  );
  const head = view.state.selection.main.head;
  const ordered = direction === 1 ? found : found.reverse();
  const target =
    ordered.find((entry) => (direction === 1 ? entry.from > head : entry.from < head)) ??
    ordered[0];
  if (!target) return false;
  view.dispatch({
    selection: { anchor: target.from },
    effects: EditorView.scrollIntoView(target.from, { y: "nearest" }),
  });
  return true;
}

/** The card of findings on the cursor's line, each with its fixes. */
function renderLineDiagnostics(view: EditorView, placed: Placed[]): HTMLElement {
  const dom = document.createElement("div");
  dom.className = "cm-tariff-diagnostics";
  for (const { diagnostic, from, to } of placed) {
    const row = document.createElement("div");
    row.className = `cm-tariff-diagnostic cm-tariff-diagnostic-${diagnostic.severity}`;
    const message = document.createElement("p");
    message.textContent = diagnostic.message;
    row.appendChild(message);
    for (const action of diagnostic.actions ?? []) {
      const button = document.createElement("button");
      button.type = "button";
      button.className = "cm-tariff-diagnostic-fix";
      button.textContent = action.name;
      button.addEventListener("mousedown", (event) => {
        event.preventDefault();
        action.apply(view, from, to);
      });
      row.appendChild(button);
    }
    dom.appendChild(row);
  }
  return dom;
}

/** Opens the findings on the cursor's line in a card (`gl` in Vim); false when it has none. */
export function showLineDiagnostics(view: EditorView): boolean {
  const line = view.state.doc.lineAt(view.state.selection.main.head);
  const placed = placedDiagnostics(view).filter(
    (entry) => entry.from <= line.to && entry.to >= line.from,
  );
  if (placed.length === 0) return false;
  openCursorTooltip(view, {
    pos: Math.max(placed[0].from, line.from),
    create: () => renderLineDiagnostics(view, placed),
  });
  return true;
}
