// lading-tools.ts is the Lading editor's tools beyond the cursor's help: signature help
// (`lading.signature`) while the cursor is inside a call, inlay values (`lading.inlays`) from the
// last run for what is on screen, Find references (Shift-F12, `lading.references`), rename (F2,
// `lading.rename`) and the refactorings (Mod-., `lading.actions`: extract a crosswalk or a
// function, promote a literal to a partner setting, move to a library). Rename and the
// refactorings can edit several files: the edits to this file are one undoable change here, and
// the rest go to the studio through the service.

import { Prec, RangeSetBuilder, StateEffect, StateField, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  WidgetType,
  keymap,
  showPanel,
  showTooltip,
  type DecorationSet,
  type Panel,
  type Tooltip,
  type ViewUpdate,
} from "@codemirror/view";
import {
  ladingActions,
  ladingDocText,
  ladingInlays,
  ladingReferences,
  ladingRename,
  ladingSignature,
  type LadingInlay,
  type LadingRange,
  type LadingSignatureHelp,
} from "@trenova/shared/edi/lading";
import { translateMessage, type CatalogMessage } from "@trenova/shared/i18n/message";
import { translate } from "@trenova/shared/i18n/runtime";
import { showEditorNotice, tariffNotices } from "../formula-editor/tariff-notice";
import {
  ladingServiceFacet,
  liveProject,
  unlessRefused,
  type LadingLanguageService,
} from "./lading-service";

/** The key that asks for the refactorings at the cursor. */
export const LADING_ACTIONS_KEY = "Mod-.";

/** The ranges the last test run did not reach in this file, marked line by line. */
export const setLadingCoverage = StateEffect.define<readonly LadingRange[]>();

const uncoveredLine = Decoration.line({ class: "cm-lading-uncovered" });

const coverageField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(marks, transaction) {
    for (const effect of transaction.effects) {
      if (!effect.is(setLadingCoverage)) continue;
      const { doc } = transaction.state;
      const lines = new Set<number>();
      for (const range of effect.value) {
        const first = doc.lineAt(Math.min(range.from, doc.length)).number;
        const last = doc.lineAt(Math.min(Math.max(range.to, range.from), doc.length)).number;
        for (let line = first; line <= last; line += 1) lines.add(line);
      }
      const builder = new RangeSetBuilder<Decoration>();
      for (const line of [...lines].sort((a, b) => a - b)) {
        const start = doc.line(line).from;
        builder.add(start, start, uncoveredLine);
      }
      return builder.finish();
    }
    return marks.map(transaction.changes);
  },
  provide: (field) => EditorView.decorations.from(field),
});

/** Tells the editor the last run's values or the other files changed, so inlays are redrawn. */
export const refreshLadingInlays = StateEffect.define<null>();

function atCursor(service: LadingLanguageService, view: EditorView, offset: number) {
  return {
    project: liveProject(service, view.state.doc.toString()),
    context: service.context(),
    path: service.path(),
    offset,
  };
}

/** The signature popup's element, the parameter being written in bold. */
export function renderLadingSignature(help: LadingSignatureHelp): HTMLElement | null {
  const signature =
    (help.active === null ? undefined : help.signatures[help.active]) ?? help.signatures[0];
  if (!signature) return null;
  const dom = document.createElement("div");
  dom.className = "cm-tariff-signature";
  const code = document.createElement("code");
  code.className = "cm-tariff-signature-code";
  const active =
    help.activeParameter === null ? undefined : signature.parameters[help.activeParameter];
  if (active) {
    const [from, to] = active.label;
    code.appendChild(document.createTextNode(signature.label.slice(0, from)));
    const bold = document.createElement("span");
    bold.className = "cm-tariff-signature-active";
    bold.textContent = signature.label.slice(from, to);
    code.appendChild(bold);
    code.appendChild(document.createTextNode(signature.label.slice(to)));
  } else {
    code.textContent = signature.label;
  }
  dom.appendChild(code);
  const doc = ladingDocText(signature.doc, (message: CatalogMessage) => translateMessage(message));
  if (doc) {
    const body = document.createElement("div");
    body.className = "cm-tariff-signature-body";
    body.textContent = doc;
    dom.appendChild(body);
  }
  return dom;
}

function signatureHelp(service: LadingLanguageService): Extension {
  return StateField.define<Tooltip | null>({
    create: () => null,
    update(value, transaction) {
      if (!transaction.docChanged && !transaction.selection) return value;
      const selection = transaction.state.selection.main;
      if (!selection.empty) return null;
      const source = transaction.state.doc.toString();
      const help = unlessRefused(
        () =>
          ladingSignature(service.edi(), {
            project: liveProject(service, source),
            context: service.context(),
            path: service.path(),
            offset: selection.head,
          }),
        null,
      );
      if (!help || help.signatures.length === 0) return null;
      return {
        pos: selection.head,
        above: true,
        strictSide: true,
        arrow: false,
        create: () => ({ dom: renderLadingSignature(help) ?? document.createElement("div") }),
      };
    },
    provide: (field) => showTooltip.from(field),
  });
}

class InlayWidget extends WidgetType {
  readonly text: string;
  readonly kind: string;

  constructor(text: string, kind: string) {
    super();
    this.text = text;
    this.kind = kind;
  }

  eq(other: InlayWidget) {
    return other.text === this.text && other.kind === this.kind;
  }

  toDOM() {
    const dom = document.createElement("span");
    dom.className = `cm-tariff-inlay cm-tariff-inlay-${this.kind === "value" ? "value" : "type"}`;
    dom.textContent = this.text;
    dom.setAttribute("aria-hidden", "true");
    return dom;
  }

  ignoreEvent() {
    return true;
  }
}

function buildInlays(view: EditorView, hints: readonly LadingInlay[]): DecorationSet {
  const length = view.state.doc.length;
  const builder = new RangeSetBuilder<Decoration>();
  for (const hint of [...hints].sort((a, b) => a.offset - b.offset)) {
    const position = Math.min(hint.offset, length);
    builder.add(
      position,
      position,
      Decoration.widget({
        widget: new InlayWidget(hint.label, hint.kind),
        side: hint.kind === "parameter" ? -1 : 1,
      }),
    );
  }
  return builder.finish();
}

function visibleInlays(service: LadingLanguageService, view: EditorView): LadingInlay[] {
  const from = view.visibleRanges[0]?.from ?? 0;
  const to = view.visibleRanges.at(-1)?.to ?? view.state.doc.length;
  return unlessRefused(
    () =>
      ladingInlays(service.edi(), {
        project: liveProject(service, view.state.doc.toString()),
        context: service.context(),
        path: service.path(),
        range: { from, to },
        values: [...service.values()],
      }),
    [],
  );
}

function inlayHints(service: LadingLanguageService): Extension {
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;

      constructor(view: EditorView) {
        this.decorations = buildInlays(view, visibleInlays(service, view));
      }

      update(update: ViewUpdate) {
        const refreshed = update.transactions.some((transaction) =>
          transaction.effects.some((effect) => effect.is(refreshLadingInlays)),
        );
        if (update.docChanged || update.viewportChanged || refreshed) {
          this.decorations = buildInlays(update.view, visibleInlays(service, update.view));
        }
      }
    },
    { decorations: (value) => value.decorations },
  );
}

type MenuItem = { label: string; detail?: string; run: (view: EditorView) => void };
type Menu = { pos: number; title: string; items: readonly MenuItem[]; active: number } | null;

const setMenu = StateEffect.define<Menu>();

const menuField = StateField.define<Menu>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) if (effect.is(setMenu)) return effect.value;
    return transaction.docChanged || transaction.selection ? null : value;
  },
  provide: (field) =>
    showTooltip.from(field, (menu): Tooltip | null =>
      menu ? { pos: menu.pos, above: false, create: (view) => renderMenu(view, menu) } : null,
    ),
});

function choose(view: EditorView, item: MenuItem) {
  view.dispatch({ effects: setMenu.of(null) });
  item.run(view);
}

function renderMenu(view: EditorView, menu: NonNullable<Menu>) {
  const dom = document.createElement("div");
  dom.className = "cm-tariff-actions";
  const list = document.createElement("ul");
  list.setAttribute("role", "menu");
  list.setAttribute("aria-label", menu.title);
  menu.items.forEach((item, index) => {
    const entry = document.createElement("li");
    entry.setAttribute("role", "none");
    const button = document.createElement("button");
    button.type = "button";
    button.setAttribute("role", "menuitem");
    button.className = "cm-tariff-action";
    if (index === menu.active) button.setAttribute("aria-current", "true");
    button.textContent = item.label;
    if (item.detail) {
      const detail = document.createElement("span");
      detail.className = "cm-lading-menu-detail";
      detail.textContent = item.detail;
      button.appendChild(detail);
    }
    button.addEventListener("mousedown", (event) => event.preventDefault());
    button.addEventListener("click", () => choose(view, item));
    entry.appendChild(button);
    list.appendChild(entry);
  });
  dom.appendChild(list);
  return { dom };
}

function moveActive(view: EditorView, by: number): boolean {
  const menu = view.state.field(menuField);
  if (!menu) return false;
  const count = menu.items.length;
  view.dispatch({ effects: setMenu.of({ ...menu, active: (menu.active + by + count) % count }) });
  return true;
}

function lineNumber(source: string, offset: number): number {
  let line = 1;
  for (let index = 0; index < Math.min(offset, source.length); index += 1) {
    if (source.charCodeAt(index) === 10) line += 1;
  }
  return line;
}

function lineText(source: string, offset: number): string {
  const start = source.lastIndexOf("\n", Math.max(offset - 1, 0)) + 1;
  const end = source.indexOf("\n", offset);
  return source.slice(start, end === -1 ? source.length : end).trim();
}

/** Lists every use of the name at the cursor, across the project; picking one opens it. */
export function findLadingReferences(view: EditorView): boolean {
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const head = view.state.selection.main.head;
  const request = atCursor(service, view, head);
  const references = unlessRefused(
    () => ladingReferences(service.edi(), { ...request, includeDeclaration: true }),
    [],
  );
  if (references.length === 0) {
    showEditorNotice(view, translate("Nothing here has references"));
    return true;
  }
  const sources = new Map(request.project.files.map((file) => [file.path, file.source]));
  const items: MenuItem[] = references.map((reference) => {
    const source = sources.get(reference.path) ?? "";
    return {
      label: translate("{0}:{1}", reference.path, lineNumber(source, reference.range.from)),
      detail: lineText(source, reference.range.from),
      run: () => service.openLocation({ path: reference.path, range: reference.range }),
    };
  });
  view.dispatch({
    effects: setMenu.of({ pos: head, title: translate("References"), items, active: 0 }),
  });
  return true;
}

/** Offers the refactorings for the selection, or says there are none here. */
export function showLadingActions(view: EditorView): boolean {
  if (view.state.readOnly) return false;
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const { from, to } = view.state.selection.main;
  const actions = unlessRefused(
    () =>
      ladingActions(service.edi(), {
        project: liveProject(service, view.state.doc.toString()),
        context: service.context(),
        path: service.path(),
        range: { from, to },
      }),
    [],
  );
  if (actions.length === 0) {
    showEditorNotice(view, translate("Nothing here can be refactored"));
    return true;
  }
  const items: MenuItem[] = actions.map((action) => ({
    label: translateMessage(action.title),
    run: () => service.applyEdits(action.edits),
  }));
  view.dispatch({
    effects: setMenu.of({ pos: from, title: translate("Refactorings"), items, active: 0 }),
  });
  return true;
}

type RenameState = { offset: number; name: string; refusal: string | null } | null;

const openRename = StateEffect.define<RenameState>();

const renameField = StateField.define<RenameState>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) if (effect.is(openRename)) return effect.value;
    return transaction.docChanged ? null : value;
  },
  provide: (field) =>
    showPanel.from(field, (state) => (state ? (view) => renamePanel(view, state) : null)),
});

function renamePanel(view: EditorView, state: NonNullable<RenameState>): Panel {
  const dom = document.createElement("form");
  dom.className = "cm-tariff-rename";
  const label = document.createElement("label");
  label.textContent = translate("Rename {0} to", state.name);
  const input = document.createElement("input");
  input.className = "cm-tariff-rename-input";
  input.value = state.name;
  input.setAttribute("aria-label", translate("New name"));
  label.appendChild(input);
  dom.appendChild(label);
  if (state.refusal) {
    const refusal = document.createElement("p");
    refusal.className = "cm-tariff-rename-refusal";
    refusal.setAttribute("role", "alert");
    refusal.textContent = state.refusal;
    dom.appendChild(refusal);
  }
  const close = () => {
    view.dispatch({ effects: openRename.of(null) });
    view.focus();
  };
  input.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      close();
    }
  });
  dom.addEventListener("submit", (event) => {
    event.preventDefault();
    const service = view.state.facet(ladingServiceFacet);
    const name = input.value.trim();
    if (!service || name === "" || name === state.name) {
      close();
      return;
    }
    const answer = unlessRefused(
      () =>
        ladingRename(service.edi(), { ...atCursor(service, view, state.offset), newName: name }),
      null,
    );
    if (!answer || "range" in answer) {
      close();
      return;
    }
    if ("error" in answer) {
      view.dispatch({
        effects: openRename.of({ ...state, refusal: translateMessage(answer.error) }),
      });
      return;
    }
    view.dispatch({ effects: openRename.of(null) });
    service.applyEdits(answer.edits);
    view.focus();
  });
  return {
    dom,
    top: true,
    mount: () => {
      input.focus();
      input.select();
    },
  };
}

/** Opens rename for the name at the cursor, or says why it cannot be renamed. */
export function startLadingRename(view: EditorView): boolean {
  if (view.state.readOnly) return false;
  const service = view.state.facet(ladingServiceFacet);
  if (!service) return false;
  const offset = view.state.selection.main.head;
  const prepared = unlessRefused(
    () => ladingRename(service.edi(), atCursor(service, view, offset)),
    null,
  );
  if (!prepared || "edits" in prepared) return false;
  view.dispatch({
    effects: openRename.of(
      "error" in prepared
        ? { offset, name: "", refusal: translateMessage(prepared.error) }
        : { offset, name: prepared.placeholder, refusal: null },
    ),
  });
  return true;
}

export function ladingTools(service: LadingLanguageService): Extension {
  return [
    tariffNotices(),
    signatureHelp(service),
    inlayHints(service),
    menuField,
    renameField,
    coverageField,
    keymap.of([
      { key: "Shift-F12", run: findLadingReferences },
      { key: "F2", run: startLadingRename },
      { key: LADING_ACTIONS_KEY, run: showLadingActions },
    ]),
    Prec.highest(
      keymap.of([
        { key: "ArrowDown", run: (view) => moveActive(view, 1) },
        { key: "ArrowUp", run: (view) => moveActive(view, -1) },
        {
          key: "Enter",
          run: (view) => {
            const menu = view.state.field(menuField);
            const item = menu?.items[menu.active];
            if (!item) return false;
            choose(view, item);
            return true;
          },
        },
        {
          key: "Escape",
          run: (view) => {
            if (!view.state.field(menuField)) return false;
            view.dispatch({ effects: setMenu.of(null) });
            return true;
          },
        },
      ]),
    ),
    EditorView.baseTheme({
      ".cm-lading-uncovered": {
        backgroundColor: "color-mix(in oklab, var(--warning) 8%, transparent)",
      },
      ".cm-lading-menu-detail": {
        display: "block",
        fontFamily: "var(--font-code)",
        fontSize: "11px",
        color: "var(--muted-foreground)",
        whiteSpace: "nowrap",
        overflow: "hidden",
        textOverflow: "ellipsis",
        maxWidth: "360px",
      },
    }),
  ];
}
