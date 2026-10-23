import { nextSnippetField } from "@codemirror/autocomplete";
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { codeMirrorSnippet, tariffCompletions } from "../tariff-complete";
import type { TariffLanguageService } from "../tariff-service";

function service(): TariffLanguageService {
  return {
    rules: rulesSync,
    kind: () => "tariff",
    context: () => EMPTY_TARIFF_CONTEXT,
    trace: () => [],
    openDefinition: vi.fn(),
    showReferences: vi.fn(),
  };
}

const views: EditorView[] = [];

afterEach(() => {
  for (const view of views.splice(0)) {
    view.dom.parentElement?.remove();
    view.destroy();
  }
});

/** Picks the completion labelled `label` at the `|` in `source`, as the list would. */
function pick(source: string, label: string): EditorView {
  const position = source.indexOf("|");
  const doc = source.replace("|", "");
  const parent = document.createElement("div");
  document.body.appendChild(parent);
  const view = new EditorView({
    state: EditorState.create({ doc, selection: { anchor: position } }),
    parent,
  });
  views.push(view);
  const result = tariffCompletions(service(), doc, position);
  const option = result?.options.find((candidate) => candidate.label === label);
  if (!option) throw new Error(`no completion ${label}`);
  const apply = option.apply as (
    view: EditorView,
    completion: unknown,
    from: number,
    to: number,
  ) => void;
  apply(view, option, result!.from, result!.to ?? position);
  return view;
}

function selected(view: EditorView): string {
  const { from, to } = view.state.selection.main;
  return view.state.sliceDoc(from, to);
}

describe("codeMirrorSnippet", () => {
  it("braces bare tab stops and keeps braced ones", () => {
    expect(codeMirrorSnippet("chosen($0)")).toBe("chosen(${0})");
    expect(codeMirrorSnippet("trip(empty: ${1:Distance}, rate: $2)")).toBe(
      "trip(empty: ${1:Distance}, rate: ${2})",
    );
    expect(codeMirrorSnippet('best "${1:Label}" {\n\t$0\n}')).toBe(
      'best "${1:Label}" {\n\t${0}\n}',
    );
  });

  it("writes escaped characters as themselves, inside a placeholder too", () => {
    expect(codeMirrorSnippet("input fee: Money = ${0:\\$0}")).toBe("input fee: Money = ${0:$0}");
    expect(codeMirrorSnippet("\\$5 \\} \\\\")).toBe("$5 } \\");
  });

  it("leaves a dollar sign that starts no tab stop alone", () => {
    expect(codeMirrorSnippet("a $ b ${x}")).toBe("a $ b ${x}");
  });
});

describe("picking a completion", () => {
  it("puts the cursor between the parentheses of a call whose parameters all have defaults", () => {
    const view = pick(
      'tariff "t"\nfn stop(fee: Money = $95) -> Money = fee\ncharge "x" = |',
      "stop",
    );
    expect(view.state.doc.toString()).toBe(
      'tariff "t"\nfn stop(fee: Money = $95) -> Money = fee\ncharge "x" = stop()',
    );
    expect(view.state.selection.main.head).toBe(view.state.doc.length - 1);
  });

  it("selects each required parameter's type in turn", () => {
    const view = pick(
      'tariff "t"\nfn trip(empty: Distance, rate: Money/mi) -> Money = rate * empty\ncharge "x" = |',
      "trip",
    );
    expect(view.state.doc.toString()).toContain(
      'charge "x" = trip(empty: Distance, rate: Money/mi)',
    );
    expect(selected(view)).toBe("Distance");
    nextSnippetField(view);
    expect(selected(view)).toBe("Money/mi");
  });

  it("offers chosen with the file's first choice label selected inside its quotes", () => {
    const view = pick(
      'tariff "t"\nbest "Linehaul" {\n  option "Flat" {\n    charge "Flat" = $900\n  }\n}\ntest "t" {\n  expect |\n}\n',
      "chosen",
    );
    expect(view.state.doc.toString()).toContain('  expect chosen("Linehaul")\n');
    expect(selected(view)).toBe("Linehaul");
  });

  it("writes the best snippet with no stray tab stop and ends in the first option", () => {
    const view = pick('tariff "t"\n\n|', "best with two options");
    expect(view.state.doc.toString()).toBe(
      'tariff "t"\n\nbest "Label" {\n  option "Per mile" {\n    \n  }\n  option "Flat" {\n    \n  }\n}',
    );
    nextSnippetField(view);
    nextSnippetField(view);
    nextSnippetField(view);
    const firstBody =
      view.state.doc.toString().indexOf('"Per mile" {\n    ') + '"Per mile" {\n    '.length;
    expect(view.state.selection.main.head).toBe(firstBody);
  });
});
