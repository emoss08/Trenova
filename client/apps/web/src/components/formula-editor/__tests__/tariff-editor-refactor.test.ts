import { nextSnippetField } from "@codemirror/autocomplete";
import { EditorState, Transaction, type Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffTest, tariffTokens } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT, type TariffCoverage } from "@trenova/shared/types/tariff";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { promoteToInput, tariffActionMenu } from "../tariff-actions";
import { tariffCompletions } from "../tariff-complete";
import {
  coverageGapLines,
  setTariffCoverage,
  showCoverageTip,
  tariffCoverage,
} from "../tariff-coverage";
import { visibleTokenDecorations } from "../tariff-highlight";
import { tariffNavigation } from "../tariff-navigate";
import { NOTICE_MS, tariffNotices } from "../tariff-notice";
import { OCCURRENCE_DELAY_MS, findReferences, tariffReferenceMarks } from "../tariff-references";
import type { TariffLanguageService } from "../tariff-service";

const SOURCE = `tariff "Regional"

input rate: Money/mi = $2.85/mi

var extra = $0
if shipment.distance > 500 mi {
  extra = $25
}
charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Flat fee" = $40
charge "Extra" = extra

test "a short haul" {
  given shipment.distance = 100 mi
  expect total == $325
}
`;

const at = (needle: string, delta = 0) => {
  const index = SOURCE.indexOf(needle);
  if (index < 0) throw new Error(`no ${needle}`);
  return index + delta;
};

function service(overrides: Partial<TariffLanguageService> = {}): TariffLanguageService {
  return {
    rules: rulesSync,
    kind: () => "tariff",
    context: () => EMPTY_TARIFF_CONTEXT,
    trace: () => [],
    openDefinition: vi.fn(),
    showReferences: vi.fn(),
    ...overrides,
  };
}

const views: EditorView[] = [];
function mount(doc: string, extensions: Extension) {
  const parent = document.createElement("div");
  document.body.appendChild(parent);
  const view = new EditorView({ state: EditorState.create({ doc, extensions }), parent });
  views.push(view);
  return view;
}

afterEach(() => {
  for (const view of views.splice(0)) {
    view.dom.parentElement?.remove();
    view.destroy();
  }
  vi.useRealTimers();
});

function marked(view: EditorView): string[] {
  const out: string[] = [];
  for (const node of view.dom.querySelectorAll(".cm-tariff-occurrence")) {
    out.push(
      `${node.textContent}:${node.classList.contains("cm-tariff-occurrence-write") ? "write" : "read"}`,
    );
  }
  return out;
}

describe("the marks at the cursor", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  it("marks every use of the name once the cursor rests, writes apart from reads", () => {
    const view = mount(SOURCE, [tariffNotices(), tariffReferenceMarks(service())]);
    view.dispatch({ selection: { anchor: at("= extra", 3) } });
    expect(marked(view)).toEqual([]);
    vi.advanceTimersByTime(OCCURRENCE_DELAY_MS);
    expect(marked(view)).toEqual(["extra:write", "extra:write", "extra:read"]);
  });

  it("clears the marks on an edit and off a name", () => {
    const view = mount(SOURCE, [tariffNotices(), tariffReferenceMarks(service())]);
    view.dispatch({ selection: { anchor: at("rate)", 1) } });
    vi.advanceTimersByTime(OCCURRENCE_DELAY_MS);
    expect(marked(view)).toEqual(["rate:write", "rate:read"]);
    view.dispatch({ changes: { from: 0, insert: " " } });
    expect(marked(view)).toEqual([]);
    view.dispatch({ selection: { anchor: 0 } });
    vi.advanceTimersByTime(OCCURRENCE_DELAY_MS);
    expect(marked(view)).toEqual([]);
  });
});

describe("find references", () => {
  it("hands the page the name and each use, in source order", () => {
    const show = vi.fn();
    const view = mount(SOURCE, [
      tariffNotices(),
      tariffReferenceMarks(service({ showReferences: show })),
    ]);
    view.dispatch({ selection: { anchor: at("extra = $25", 2) } });
    expect(findReferences(view)).toBe(true);
    expect(show).toHaveBeenCalledWith({
      name: "extra",
      offset: at("extra = $25", 2),
      references: [
        {
          range: { from: at("extra = $0"), to: at("extra = $0") + 5 },
          kind: "declaration",
          module: null,
        },
        {
          range: { from: at("extra = $25"), to: at("extra = $25") + 5 },
          kind: "write",
          module: null,
        },
        { range: { from: at("= extra", 2), to: at("= extra", 7) }, kind: "read", module: null },
      ],
    });
  });

  it("says so at the cursor off a name, and the notice goes on its own", () => {
    vi.useFakeTimers();
    const show = vi.fn();
    const view = mount(SOURCE, [
      tariffNotices(),
      tariffReferenceMarks(service({ showReferences: show })),
    ]);
    view.dispatch({ selection: { anchor: 1 } });
    expect(findReferences(view)).toBe(true);
    expect(show).not.toHaveBeenCalled();
    expect(view.dom.querySelector(".cm-tariff-notice")?.textContent).toBe(
      "Put the cursor on a name to find its references",
    );
    vi.advanceTimersByTime(NOTICE_MS);
    expect(view.dom.querySelector(".cm-tariff-notice")).toBeNull();
  });
});

describe("promote to input", () => {
  const extensions = (extra: Extension = []) => [
    extra,
    tariffNotices(),
    tariffNavigation(service()),
    tariffActionMenu(),
  ];

  it("offers the module's refactorings for the literal at the cursor", () => {
    const view = mount(SOURCE, extensions());
    view.dispatch({ selection: { anchor: at("$40", 1) } });
    expect(promoteToInput(view)).toBe(true);
    const items = [...view.dom.querySelectorAll(".cm-tariff-action")].map(
      (node) => node.textContent,
    );
    expect(items).toEqual(["Promote $40 to an input"]);
    expect(view.dom.querySelector(".cm-tariff-action code")?.textContent).toBe("$40");
  });

  it("applies the chosen action as one undoable change and opens rename on the new input", () => {
    const changes: string[] = [];
    const view = mount(
      SOURCE,
      extensions(
        EditorView.updateListener.of((update) => {
          for (const transaction of update.transactions) {
            if (transaction.docChanged)
              changes.push(String(transaction.annotation(Transaction.userEvent)));
          }
        }),
      ),
    );
    view.dispatch({ selection: { anchor: at("$40", 1) } });
    promoteToInput(view);
    view.dom.querySelector<HTMLButtonElement>(".cm-tariff-action")!.click();
    const text = view.state.doc.toString();
    expect(text).toContain("input rate: Money/mi = $2.85/mi\ninput flatFeeAmount: Money = $40\n");
    expect(text).toContain('charge "Flat fee" = flatFeeAmount\n');
    expect(view.dom.querySelector(".cm-tariff-rename-refusal")?.textContent ?? null).toBeNull();
    expect(view.dom.querySelector(".cm-tariff-actions")).toBeNull();
    expect(view.dom.querySelector<HTMLInputElement>(".cm-tariff-rename-input")?.value).toBe(
      "flatFeeAmount",
    );
    expect(document.activeElement).toBe(view.dom.querySelector(".cm-tariff-rename-input"));
    expect(changes).toEqual(["refactor"]);
  });

  it("moves through the offers with the arrow keys and closes on Escape", () => {
    const source = 'tariff "t"\ncharge "a" = $5\ncharge "b" = $5\n';
    const view = mount(source, extensions());
    view.dispatch({ selection: { anchor: source.indexOf("$5") + 1 } });
    promoteToInput(view);
    const current = () =>
      view.dom.querySelector(".cm-tariff-action[aria-current='true']")?.textContent;
    expect(current()).toBe("Promote $5 to an input");
    view.contentDOM.dispatchEvent(
      new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }),
    );
    expect(current()).toBe("Promote all 2 uses of $5 to an input");
    view.contentDOM.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
    expect(view.state.doc.toString()).toBe(
      'tariff "t"\n\ninput aAmount: Money = $5\n\ncharge "a" = aAmount\ncharge "b" = aAmount\n',
    );
  });

  it("says there is nothing to promote and changes nothing", () => {
    const view = mount(SOURCE, extensions());
    view.dispatch({ selection: { anchor: 1 } });
    expect(promoteToInput(view)).toBe(true);
    expect(view.dom.querySelector(".cm-tariff-rename-refusal")?.textContent ?? null).toBeNull();
    expect(view.dom.querySelector(".cm-tariff-actions")).toBeNull();
    expect(view.dom.querySelector(".cm-tariff-notice")?.textContent).toBe(
      "Nothing here can be promoted to an input",
    );
    expect(view.state.doc.toString()).toBe(SOURCE);
  });

  it("offers nothing in a read-only editor", () => {
    const view = mount(SOURCE, [...extensions(), EditorState.readOnly.of(true)]);
    view.dispatch({ selection: { anchor: at("$40", 1) } });
    expect(promoteToInput(view)).toBe(false);
  });
});

describe("coverage", () => {
  const coverage = (): TariffCoverage => {
    const run = tariffTest(rulesSync(), { source: SOURCE, context: EMPTY_TARIFF_CONTEXT });
    return run.coverage!;
  };

  it("marks the lines of the regions no test reaches, with what they are", () => {
    const view = mount(SOURCE, tariffCoverage());
    setTariffCoverage(view, coverage());
    const lines = coverageGapLines(view.state);
    const ifLine = view.state.doc.lineAt(at("  extra = $25")).number;
    expect(lines.map((gap) => [gap.line, gap.kind])).toEqual([
      [ifLine - 1, "if"],
      [ifLine, "if"],
      [ifLine + 1, "if"],
    ]);
    const markers = view.dom.querySelectorAll(".cm-tariff-coverage-gap");
    expect(markers).toHaveLength(3);
    expect(markers[0]?.getAttribute("aria-label")).toBe("No test reaches this branch of the if");
  });

  it("draws no gutter until there is coverage, and takes it away again", () => {
    const view = mount(SOURCE, tariffCoverage());
    expect(view.dom.querySelector(".cm-tariff-coverage-gutter")).toBeNull();
    setTariffCoverage(view, coverage());
    expect(view.dom.querySelector(".cm-tariff-coverage-gutter")).not.toBeNull();
    setTariffCoverage(view, null);
    expect(view.dom.querySelector(".cm-tariff-coverage-gutter")).toBeNull();
    expect(coverageGapLines(view.state)).toEqual([]);
  });

  it("keeps the marks on their lines through an edit above them", () => {
    const view = mount(SOURCE, tariffCoverage());
    setTariffCoverage(view, coverage());
    const before = coverageGapLines(view.state).map((gap) => gap.line);
    view.dispatch({ changes: { from: 0, insert: "// a comment\n" } });
    expect(coverageGapLines(view.state).map((gap) => gap.line)).toEqual(
      before.map((line) => line + 1),
    );
  });
});

describe("the new keywords and snippets", () => {
  const NEW = `tariff "LTL"
input rate: Money/lb = $0.31/lb where $0.05/lb..=$2/lb
best "Linehaul" {
  option "Flat" {
    charge "Flat" = $900
  }
  option "Weight" {
    charge "Weight" = (shipment.weight ?? 0 lb) * rate
  }
}
property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  assume shipment.distance > 0 mi
  expect total nondecreasing in shipment.weight
}
`;

  it("colours where, best, option, property, vary, assume and nondecreasing as keywords", () => {
    const tokens = tariffTokens(rulesSync(), NEW);
    const keywords: string[] = [];
    visibleTokenDecorations(tokens, [{ from: 0, to: NEW.length }]).between(
      0,
      NEW.length,
      (from, to, deco) => {
        if (String(deco.spec.class).includes("cm-syntax-keyword"))
          keywords.push(NEW.slice(from, to));
      },
    );
    for (const word of ["where", "best", "option", "property", "vary", "assume", "nondecreasing"]) {
      expect(keywords).toContain(word);
    }
  });

  it("expands the best snippet with a tab stop for the label and each option", () => {
    const source = 'tariff "t"\n\n';
    const view = mount(source, []);
    const result = tariffCompletions(service(), source, source.length);
    const best = result?.options.find((option) => option.label === "best with two options");
    (best!.apply as (view: EditorView, completion: unknown, from: number, to: number) => void)(
      view,
      best,
      result!.from,
      result!.to ?? source.length,
    );
    const selected = () =>
      view.state.sliceDoc(view.state.selection.main.from, view.state.selection.main.to);
    expect(view.state.doc.toString()).toContain('best "Label" {');
    expect(selected()).toBe("Label");
    nextSnippetField(view);
    expect(selected()).toBe("Per mile");
    nextSnippetField(view);
    expect(selected()).toBe("Flat");
  });

  it("expands the vary snippet inside a property with its target and range", () => {
    const source = 'tariff "t"\ncharge "a" = $1\nproperty "p" {\n  \n}\n';
    const position = source.indexOf("{\n  ") + 4;
    const view = mount(source, []);
    const result = tariffCompletions(service(), source, position);
    const vary = result?.options.find((option) => option.label === "vary … in …");
    (vary!.apply as (view: EditorView, completion: unknown, from: number, to: number) => void)(
      view,
      vary,
      result!.from,
      result!.to ?? position,
    );
    const selected = () =>
      view.state.sliceDoc(view.state.selection.main.from, view.state.selection.main.to);
    expect(view.state.doc.toString()).toContain("vary shipment.weight in 1 lb..40_000 lb");
    expect(selected()).toBe("shipment.weight");
    nextSnippetField(view);
    expect(selected()).toBe("1 lb..40_000 lb");
  });

  it("completes property as a keyword at the top of a file", () => {
    const source = 'tariff "t"\ncharge "a" = $1\npro';
    const result = tariffCompletions(service(), source, source.length);
    expect(result?.options.map((option) => option.label)).toContain("property");
  });
});

describe("the coverage tooltip", () => {
  it("says, over a marked line, that no test reaches it and what it is", () => {
    const run = tariffTest(rulesSync(), { source: SOURCE, context: EMPTY_TARIFF_CONTEXT });
    const view = mount(SOURCE, tariffCoverage());
    setTariffCoverage(view, run.coverage!);
    // The gutter finds the hovered line from the pointer's height, which jsdom does not lay out,
    // so the line is given here as the gutter's handler gives it.
    const [first] = coverageGapLines(view.state);
    showCoverageTip(view, first!.from);
    expect(view.dom.querySelector(".cm-tariff-coverage-tip")?.textContent).toBe(
      "No test reaches this branch of the if",
    );
    showCoverageTip(view, 0);
    expect(view.dom.querySelector(".cm-tariff-coverage-tip")).toBeNull();
    showCoverageTip(view, first!.from);
    showCoverageTip(view, null);
    expect(view.dom.querySelector(".cm-tariff-coverage-tip")).toBeNull();
  });
});
