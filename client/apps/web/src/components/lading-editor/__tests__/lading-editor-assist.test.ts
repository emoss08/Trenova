import { setDiagnostics } from "@codemirror/lint";
import { EditorSelection, EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import type { LadingDiagnostic, LadingProjectInput } from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ladingCompletions, opensLadingCompletion } from "../lading-complete";
import { toLadingLint } from "../lading-lint";
import { checkProject, liveProject, type LadingLanguageService } from "../lading-service";
import { ladingEditorStatus } from "../lading-status-bar";

const PATH = "maps/tender-in.lading";
const SAVED = 'map tenderIn from 204 to Tender\n\nbol <- L11[02 == "BM"].01\n';

const PROJECT: LadingProjectInput = {
  files: [
    { path: "partner.lading", source: 'partner acme "Acme"\n' },
    { path: PATH, source: SAVED },
  ],
  samples: [],
};

function service(call: (op: string, input: unknown) => unknown): LadingLanguageService {
  const edi = { call: vi.fn(call) } as unknown as EdiRuntime;
  return {
    edi: () => edi,
    project: () => PROJECT,
    context: () => ({ org: { name: "Trenova", scac: "TRNV", zone: "America/Chicago" } }),
    path: () => PATH,
    values: () => [],
    openLocation: vi.fn(),
    applyEdits: vi.fn(),
  };
}

const views: EditorView[] = [];

afterEach(() => {
  for (const view of views.splice(0)) {
    view.dom.parentElement?.remove();
    view.destroy();
  }
});

function mount(doc: string, anchor = 0): EditorView {
  const parent = document.createElement("div");
  document.body.appendChild(parent);
  const view = new EditorView({
    state: EditorState.create({ doc, selection: { anchor } }),
    parent,
  });
  views.push(view);
  return view;
}

describe("liveProject", () => {
  it("is the studio's project while it holds the editor's text", () => {
    const languageService = service(() => null);
    expect(liveProject(languageService, SAVED)).toBe(PROJECT);
  });

  it("swaps in the editor's text and gives the same object for the same text", () => {
    const languageService = service(() => null);
    const typed = `${SAVED}po <- `;
    const first = liveProject(languageService, typed);
    expect(first.files).toEqual([PROJECT.files[0], { path: PATH, source: typed }]);
    expect(liveProject(languageService, typed)).toBe(first);
    expect(liveProject(languageService, `${typed}L`)).not.toBe(first);
  });
});

describe("checkProject", () => {
  it("checks the same project again when a new context arrives, and once per context", () => {
    const project: LadingProjectInput = { files: [{ path: PATH, source: SAVED }], samples: [] };
    const empty = {
      files: [],
      partner: null,
      maps: [],
      flows: [],
      rules: [],
      libraries: [],
      guides: [],
      errors: 0,
      warnings: 0,
    };
    const call = vi.fn((_op: string, _input: { context: unknown }) => empty);
    const edi = { call } as unknown as EdiRuntime;
    const before = { org: { name: "Trenova", scac: "TRNV", zone: "America/Chicago" } };
    const after = {
      ...before,
      customFields: { shipment: [{ name: "po_number", type: "Text", label: "PO number" }] },
    };
    checkProject(edi, project, before);
    checkProject(edi, project, before);
    checkProject(edi, project, after);
    expect(call.mock.calls.map(([, input]) => input.context)).toEqual([before, after]);
  });
});

describe("ladingCompletions", () => {
  it("asks about the text the editor holds, not the one the studio last saw", () => {
    const typed = `${SAVED}po <- L`;
    const asked: { op: string; input: { project: LadingProjectInput; offset: number } }[] = [];
    const languageService = service((op, input) => {
      asked.push({ op, input: input as { project: LadingProjectInput; offset: number } });
      return { range: { from: typed.length - 1, to: typed.length }, items: [] };
    });
    ladingCompletions(languageService, typed, typed.length);
    expect(asked.map((request) => request.op)).toEqual(["lading.complete"]);
    expect(asked[0]?.input.offset).toBe(typed.length);
    expect(asked[0]?.input.project.files[1]).toEqual({ path: PATH, source: typed });
  });

  it("keeps the module's order, shows each kind's icon and asks again after a dot", () => {
    const source = "bol <- L";
    const languageService = service(() => ({
      range: { from: 7, to: 8 },
      items: [
        {
          label: "L11",
          kind: "segment",
          detail: "Reference",
          insert: "L11",
          snippet: false,
          sort: "0",
        },
        {
          label: "lookup",
          kind: "keyword",
          detail: "",
          insert: "lookup",
          snippet: false,
          sort: "5",
        },
      ],
    }));
    const result = ladingCompletions(languageService, source, 8);
    expect(result?.from).toBe(7);
    expect(result?.options.map((option) => [option.label, option.type])).toEqual([
      ["L11", "class"],
      ["lookup", "keyword"],
    ]);
    expect(result!.options[0]!.boost!).toBeGreaterThan(result!.options[1]!.boost!);
    const validFor = result?.validFor as RegExp;
    expect(validFor.test("L11")).toBe(true);
    expect(validFor.test("L11.")).toBe(false);
  });

  it("filters codes inside quotes by anything but the closing quote", () => {
    const source = 'x <- L11[02 == "B"]';
    const quote = source.indexOf('"') + 1;
    const languageService = service(() => ({
      range: { from: quote, to: quote + 1 },
      items: [
        {
          label: "BM",
          kind: "constant",
          detail: "Bill of Lading Number",
          insert: "BM",
          snippet: false,
          sort: "00000",
        },
      ],
    }));
    const result = ladingCompletions(languageService, source, quote + 1);
    expect(result?.from).toBe(quote);
    expect(result?.to).toBe(quote + 1);
    const validFor = result?.validFor as RegExp;
    expect(validFor.test("B M")).toBe(true);
    expect(validFor.test('B"')).toBe(false);
  });
});

describe("opensLadingCompletion", () => {
  it.each([
    ["bol <- L", true],
    ["stop.", true],
    ['L11[02 == "', true],
    ["bol <- ", false],
    ["", false],
    ["x == (", false],
  ])("opens after %j: %s", (before, opens) => {
    expect(opensLadingCompletion(before)).toBe(opens);
  });
});

describe("toLadingLint", () => {
  const diagnostic: LadingDiagnostic = {
    range: { from: 4, to: 99 },
    severity: "error",
    code: "flow-receive-taken",
    message: { message: "Flow {0} already receives {1}", args: ["tenders", "204"] },
    fixes: [
      { label: { message: "Remove it" }, changes: [{ range: { from: 0, to: 4 }, insert: "" }] },
    ],
    related: [{ range: { from: 10, to: 13 }, message: { message: "Received here" } }],
  };

  it("clamps the range and offers the fixes", () => {
    const lint = toLadingLint(diagnostic, 20);
    expect(lint).toMatchObject({
      from: 4,
      to: 20,
      severity: "error",
      message: "Flow tenders already receives 204",
    });
    expect(lint.actions?.map((action) => action.name)).toEqual(["Remove it"]);
  });

  it("shows the code once and links each related place, which moves the cursor there", () => {
    const view = mount("first line\nsecond line\n");
    const lint = toLadingLint(diagnostic, view.state.doc.length);
    const dom = lint.renderMessage!(view) as HTMLElement;
    expect(dom.textContent?.split("flow-receive-taken")).toHaveLength(2);
    expect(lint.source).toBeUndefined();
    const link = dom.querySelector("button");
    expect(link?.textContent).toBe("Line 1: Received here");
    link!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    expect(view.state.selection.main.from).toBe(10);
    expect(view.state.selection.main.to).toBe(13);
  });
});

describe("ladingEditorStatus", () => {
  it("gives the cursor's line and column and counts errors and warnings", () => {
    const view = mount("one\ntwo three\n", 8);
    view.dispatch(
      setDiagnostics(view.state, [
        { from: 0, to: 3, severity: "error", message: "a" },
        { from: 4, to: 7, severity: "error", message: "b" },
        { from: 8, to: 13, severity: "warning", message: "c" },
        { from: 8, to: 13, severity: "info", message: "d" },
      ]),
    );
    expect(ladingEditorStatus(view.state)).toEqual({ line: 2, column: 5, errors: 2, warnings: 1 });
    view.dispatch({ selection: EditorSelection.cursor(0) });
    expect(ladingEditorStatus(view.state)).toMatchObject({ line: 1, column: 1 });
  });
});
