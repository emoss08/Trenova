import { setDiagnostics, type Diagnostic } from "@codemirror/lint";
import { EditorState, type Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { getCM, Vim, type CodeMirrorV } from "@replit/codemirror-vim";
import { rulesSync } from "@trenova/shared/rules/runtime";
import type { TariffContext } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { hasCursorTooltip } from "../cursor-tooltip";
import { EDITOR_SAVE_EVENT, tariffEditorCommands, tariffLanguageData } from "../tariff-commands";
import { tariffHoverCard } from "../tariff-hover";
import { tariffNavigation } from "../tariff-navigate";
import type { TariffLanguageService } from "../tariff-service";
import { tariffVim } from "../tariff-vim";

const TARIFF = `tariff "Regional per mile"

/// The rate per loaded mile.
input rate: Money/mi = $2.85/mi

let linehaul = rate * shipment.distance
charge "Linehaul" = linehaul
`;

const CONTEXT: TariffContext = { libraries: [], tables: [], codes: null, settings: {} };

function service(): TariffLanguageService {
  return {
    rules: rulesSync,
    kind: () => "tariff",
    context: () => CONTEXT,
    trace: () => [],
    openDefinition: vi.fn(),
  };
}

const views: EditorView[] = [];

function mount(doc: string, extensions: Extension[] = []) {
  const languageService = service();
  const parent = document.createElement("div");
  document.body.appendChild(parent);
  const view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [
        tariffVim(),
        tariffLanguageData,
        tariffNavigation(languageService),
        tariffHoverCard(languageService),
        ...extensions,
      ],
    }),
    parent,
  });
  views.push(view);
  return view;
}

function vimOf(view: EditorView): CodeMirrorV {
  const cm = getCM(view);
  if (!cm) throw new Error("Vim is not attached");
  return cm as unknown as CodeMirrorV;
}

/**
 * Presses keys as Vim reads them. A character Vim leaves alone in insert mode is typed, as the
 * browser would type it.
 */
function press(view: EditorView, ...keys: string[]) {
  const cm = vimOf(view);
  for (const key of keys) {
    const handled = Vim.handleKey(cm, key, "user");
    if (!handled && key.length === 1 && cm.state.vim.insertMode) {
      view.dispatch(view.state.replaceSelection(key));
    }
  }
}

function mode(view: EditorView): "insert" | "normal" | "visual" {
  const { vim } = vimOf(view).state;
  if (vim.insertMode) return "insert";
  return vim.visualMode ? "visual" : "normal";
}

const cursorAt = (view: EditorView, position: number) =>
  view.dispatch({ selection: { anchor: position } });

const head = (view: EditorView) => view.state.selection.main.head;

afterEach(() => {
  for (const view of views.splice(0)) {
    view.dom.parentElement?.remove();
    view.destroy();
  }
});

describe("leaving insert mode", () => {
  it.each(["jj", "jk"])("leaves insert mode on %s without typing it", (keys) => {
    const view = mount("let x = 1\n");
    press(view, "A", "+", "2", ...keys.split(""));
    expect(view.state.doc.toString()).toBe("let x = 1+2\n");
    expect(mode(view)).toBe("normal");
  });

  it("keeps a j that another key follows", () => {
    const view = mount("x\n");
    press(view, "A", "j", "o", "b");
    expect(view.state.doc.toString()).toBe("xjob\n");
    expect(mode(view)).toBe("insert");
  });
});

describe("leader keys", () => {
  it("makes Space a leader instead of a move right", () => {
    const view = mount("let a = 1\n");
    cursorAt(view, 0);
    press(view, "<Space>");
    expect(head(view)).toBe(0);
  });

  it("formats with Space l f through the editor's own format command", () => {
    const format = vi.fn(() => true);
    const view = mount("let a = 1\n", [tariffEditorCommands.of({ format })]);
    press(view, "<Space>", "l", "f");
    expect(format).toHaveBeenCalledWith(view);
  });

  it("asks the page to save on Space w and on :w", () => {
    const view = mount("let a = 1\n");
    const save = vi.fn();
    window.addEventListener(EDITOR_SAVE_EVENT, save);
    press(view, "<Space>", "w");
    Vim.handleEx(vimOf(view), "w");
    window.removeEventListener(EDITOR_SAVE_EVENT, save);
    expect(save).toHaveBeenCalledTimes(2);
  });

  it("opens rename on Space l r with the name under the cursor", () => {
    const view = mount(TARIFF);
    cursorAt(view, TARIFF.indexOf("linehaul =") + 2);
    press(view, "<Space>", "l", "r");
    const input = view.dom.querySelector<HTMLInputElement>(".cm-tariff-rename-input");
    expect(input?.value).toBe("linehaul");
  });
});

describe("comments", () => {
  it("comments and uncomments the line with gcc, using Tariff's //", () => {
    const view = mount("let a = 1\nlet b = 2\n");
    cursorAt(view, 0);
    press(view, "g", "c", "c");
    expect(view.state.doc.toString()).toBe("// let a = 1\nlet b = 2\n");
    press(view, "g", "c", "c");
    expect(view.state.doc.toString()).toBe("let a = 1\nlet b = 2\n");
  });

  it("comments a visual selection's lines with Space / and returns to normal mode", () => {
    const view = mount("let a = 1\nlet b = 2\nlet c = 3\n");
    cursorAt(view, 0);
    press(view, "V", "j", "<Space>", "/");
    expect(view.state.doc.toString()).toBe("// let a = 1\n// let b = 2\nlet c = 3\n");
    expect(mode(view)).toBe("normal");
  });
});

describe("language features", () => {
  it("goes to a declaration on gd and stays in normal mode at its start", () => {
    const view = mount(TARIFF);
    cursorAt(view, TARIFF.indexOf("linehaul\n") + 2);
    press(view, "g", "d");
    expect(head(view)).toBe(TARIFF.indexOf("linehaul ="));
    expect(view.state.selection.main.empty).toBe(true);
    expect(mode(view)).toBe("normal");
  });

  it("opens the hover card at the cursor on K and closes it when the cursor moves", () => {
    const view = mount(TARIFF);
    cursorAt(view, TARIFF.indexOf("rate *") + 1);
    press(view, "K");
    expect(hasCursorTooltip(view)).toBe(true);
    expect(view.dom.querySelector(".cm-tariff-hover")?.textContent).toContain("rate");
    press(view, "w");
    expect(hasCursorTooltip(view)).toBe(false);
  });
});

describe("diagnostics", () => {
  const SOURCE = "aaaa\nbbbb\ncccc\ndddd\n";
  const findings: Diagnostic[] = [
    { from: 0, to: 4, severity: "warning", message: "First warning" },
    { from: 5, to: 9, severity: "error", message: "An error" },
    {
      from: 10,
      to: 14,
      severity: "warning",
      message: "Second warning",
      actions: [{ name: "Fix it", apply: () => undefined }],
    },
  ];

  function withFindings() {
    const view = mount(SOURCE);
    view.dispatch(setDiagnostics(view.state, findings));
    cursorAt(view, 0);
    return view;
  }

  it("moves to the next and previous finding on ]d and [d, wrapping at the ends", () => {
    const view = withFindings();
    press(view, "]", "d");
    expect(head(view)).toBe(5);
    press(view, "]", "d");
    expect(head(view)).toBe(10);
    press(view, "]", "d");
    expect(head(view)).toBe(0);
    press(view, "[", "d");
    expect(head(view)).toBe(10);
  });

  it("moves between errors only on ]e and warnings only on ]w", () => {
    const view = withFindings();
    press(view, "]", "e");
    expect(head(view)).toBe(5);
    press(view, "]", "e");
    expect(head(view)).toBe(5);
    press(view, "]", "w");
    expect(head(view)).toBe(10);
    press(view, "[", "w");
    expect(head(view)).toBe(0);
  });

  it("shows the cursor line's findings with their fixes on gl", () => {
    const view = withFindings();
    cursorAt(view, 11);
    press(view, "g", "l");
    const card = view.dom.querySelector(".cm-tariff-diagnostics");
    expect(card?.querySelector("p")?.textContent).toBe("Second warning");
    expect(card?.querySelector(".cm-tariff-diagnostic-warning")).not.toBeNull();
    expect(card?.querySelector("button")?.textContent).toBe("Fix it");
  });

  it("opens nothing on gl when the line has no finding", () => {
    const view = withFindings();
    cursorAt(view, 16);
    press(view, "g", "l");
    expect(hasCursorTooltip(view)).toBe(false);
  });
});
