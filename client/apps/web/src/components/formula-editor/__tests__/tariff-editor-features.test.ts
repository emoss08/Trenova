import { EditorState, type Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { RulesRequestError, rulesSync } from "@trenova/shared/rules/runtime";
import {
  tariffCheck,
  tariffHover,
  tariffSignature,
  tariffTokens,
} from "@trenova/shared/rules/tariff";
import type { TariffContext } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { tariffCompletions } from "../tariff-complete";
import { foldAt } from "../tariff-fold";
import { formatTariffDocument } from "../tariff-format";
import { visibleTokenDecorations } from "../tariff-highlight";
import { renderTariffHover } from "../tariff-hover";
import { tariffInlayHints } from "../tariff-inlays";
import { toTariffLint } from "../tariff-lint";
import { goToDefinition, startRename, tariffNavigation } from "../tariff-navigate";
import { checkTariffSource, type TariffLanguageService } from "../tariff-service";
import { renderTariffSignature } from "../tariff-signature";

const LIBRARY = `library "org.accessorials"

/// Extra stops: \`fee\` for every stop beyond the first \`included\`.
fn extraStops(stops: Int, included: Int = 2, fee: Money = $75) -> Money {
  if stops <= included {
    return $0
  }
  return fee * (stops - included)
}
`;

const TARIFF = `tariff "Regional per mile"

use org.accessorials.{ extraStops }

/// The rate per loaded mile.
input rate: Money/mi = $2.85/mi

let linehaul = rate * shipment.distance
charge "Linehaul" = linehaul
charge "Extra stops" = extraStops(shipment.stops.len)

test "a long haul" {
  given shipment.distance = 100 mi
  expect charge("Linehaul") == $285
}
`;

const CONTEXT: TariffContext = {
  libraries: [{ path: "org.accessorials", version: "flv_1", source: LIBRARY }],
  tables: [],
  codes: null,
  settings: {},
};

const at = (needle: string, delta = 0) => {
  const index = TARIFF.indexOf(needle);
  if (index < 0) throw new Error(`no ${needle}`);
  return index + delta;
};

function service(overrides: Partial<TariffLanguageService> = {}): TariffLanguageService {
  return {
    rules: rulesSync,
    kind: () => "tariff",
    context: () => CONTEXT,
    trace: () => [],
    openDefinition: vi.fn(),
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
});

describe("completion", () => {
  it("offers a record's fields after a dot, in the module's order, over the module's range", () => {
    const position = at("shipment.distance\n") + "shipment.".length;
    const result = tariffCompletions(service(), TARIFF, position);
    expect(result).not.toBeNull();
    expect(result?.from).toBe(position);
    const labels = result?.options.map((option) => option.label) ?? [];
    expect(labels).toContain("distance");
    expect(labels).toContain("stops");
    const boosts = result?.options.map((option) => option.boost ?? 0) ?? [];
    expect([...boosts].sort((a, b) => b - a)).toEqual(boosts);
  });

  it("rewrites the dot to ?. when a pick is on an optional value", () => {
    const source = 'tariff "x"\nlet w = shipment.weight.\n';
    const position = source.indexOf("weight.") + "weight.".length;
    const view = mount(source, []);
    const result = tariffCompletions(service(), source, position);
    const value = result?.options.find((option) => option.label === "value");
    expect(typeof value?.apply).toBe("function");
    (value!.apply as (view: EditorView, completion: unknown, from: number, to: number) => void)(
      view,
      value,
      result!.from,
      result!.to ?? position,
    );
    expect(view.state.doc.toString()).toBe('tariff "x"\nlet w = shipment.weight?.value\n');
  });

  it("offers nothing, and logs once, when the module refuses the request", () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const refusing = service({
      rules: () =>
        ({
          call: (op: string) => {
            throw new RulesRequestError(op, "invalid_input", "unknown field");
          },
        }) as unknown as ReturnType<typeof rulesSync>,
    });
    expect(tariffCompletions(refusing, TARIFF, 0)).toBeNull();
    expect(tariffCompletions(refusing, TARIFF, 1)).toBeNull();
    expect(error).toHaveBeenCalledTimes(1);
    error.mockRestore();
  });
});

describe("hover", () => {
  it("shows a library function's declaration as code and its doc", () => {
    const hover = tariffHover(rulesSync(), {
      source: TARIFF,
      context: CONTEXT,
      kind: "tariff",
      offset: at("extraStops(shipment", 2),
    });
    expect(hover?.contents.signature).toBe(
      "fn extraStops(stops: Int, included: Int = 2, fee: Money = $75) -> Money",
    );
    const dom = renderTariffHover(hover!, rulesSync());
    expect(dom.querySelector(".cm-tariff-hover-title")?.textContent).toBe(
      hover?.contents.signature,
    );
    expect(dom.querySelector(".cm-tariff-hover-title .cm-syntax-keyword")?.textContent).toBe("fn");
    expect(dom.querySelector(".cm-tariff-hover-body")?.textContent).toBe(
      "Extra stops: fee for every stop beyond the first included.",
    );
    expect(dom.querySelectorAll(".cm-tariff-hover-body code")).toHaveLength(2);
  });

  it("says what a binding came to in the last preview", () => {
    const dom = renderTariffHover(
      {
        range: { from: 0, to: 1 },
        contents: {
          signature: "let linehaul: Money",
          type: "Money",
          unit: "$",
          doc: null,
          value: { value: { $q: "285", u: "$" }, display: "$285.00", times: 1 },
        },
      },
      rulesSync(),
    );
    expect(dom.querySelector(".cm-tariff-hover-value")?.textContent).toBe(
      "In the preview: $285.00",
    );
  });
});

describe("signature help", () => {
  it("bolds the parameter being written", () => {
    const help = tariffSignature(rulesSync(), {
      source: TARIFF,
      context: CONTEXT,
      kind: "tariff",
      offset: at("extraStops(shipment") + "extraStops(".length,
    });
    const dom = renderTariffSignature(help!);
    expect(dom?.querySelector(".cm-tariff-signature-active")?.textContent).toBe("stops: Int");
    expect(dom?.querySelector(".cm-tariff-signature-code")?.textContent).toBe(
      help?.signatures[0]?.label,
    );
  });

  it("falls back to the first signature when the module names none active", () => {
    const dom = renderTariffSignature({
      signatures: [{ label: "f(x: Int)", doc: null, parameters: [] }],
      active: null,
      activeParameter: null,
    });
    expect(dom?.textContent).toBe("f(x: Int)");
  });
});

describe("go to definition", () => {
  it("moves to a declaration in this file", () => {
    const languageService = service();
    const view = mount(TARIFF, [tariffNavigation(languageService)]);
    expect(goToDefinition(languageService, view, at('linehaul\ncharge "Extra', 2))).toBe(true);
    const declared = at("linehaul =");
    expect(view.state.selection.main.from).toBe(declared);
    expect(view.state.selection.main.to).toBe(declared + "linehaul".length);
    expect(languageService.openDefinition).not.toHaveBeenCalled();
  });

  it("opens an imported library's declaration with the name that was followed", () => {
    const languageService = service();
    const view = mount(TARIFF, [tariffNavigation(languageService)]);
    expect(goToDefinition(languageService, view, at("extraStops(shipment", 2))).toBe(true);
    const declared = LIBRARY.indexOf("extraStops");
    expect(languageService.openDefinition).toHaveBeenCalledWith({
      module: "org.accessorials",
      range: { from: declared, to: declared + "extraStops".length },
      name: "extraStops",
    });
  });
});

describe("rename", () => {
  it("renames every reference the checker resolved, as one undoable change", () => {
    const view = mount(TARIFF, [tariffNavigation(service())]);
    view.dispatch({ selection: { anchor: at("linehaul =", 2) } });
    expect(startRename(view)).toBe(true);
    const input = view.dom.querySelector<HTMLInputElement>(".cm-tariff-rename-input");
    expect(input?.value).toBe("linehaul");
    input!.value = "haul";
    input!.form!.dispatchEvent(new Event("submit", { cancelable: true }));
    const text = view.state.doc.toString();
    expect(text).toContain("let haul = rate * shipment.distance");
    expect(text).toContain('charge "Linehaul" = haul\n');
    expect(view.dom.querySelector(".cm-tariff-rename")).toBeNull();
  });

  it("says why an imported name cannot be renamed here", () => {
    const view = mount(TARIFF, [tariffNavigation(service())]);
    view.dispatch({ selection: { anchor: at("extraStops(shipment", 2) } });
    expect(startRename(view)).toBe(true);
    expect(view.dom.querySelector(".cm-tariff-rename-refusal")?.textContent).toBe(
      "`extraStops` is declared in the library `org.accessorials`; rename it there, then update this file's import",
    );
  });
});

describe("inlay hints", () => {
  it("draws the inferred type after a binding", () => {
    const view = mount(TARIFF, [tariffInlayHints(service())]);
    const hints = [...view.dom.querySelectorAll(".cm-tariff-inlay")].map(
      (hint) => hint.textContent,
    );
    expect(hints).toContain(": Money");
  });
});

describe("folding", () => {
  it("folds a test block from the outline, keeping its braces", () => {
    const check = checkTariffSource(rulesSync(), TARIFF, "tariff", CONTEXT);
    const open = at('test "a long haul" {');
    const lineEnd = TARIFF.indexOf("\n", open);
    const fold = foldAt(check.outline?.folding ?? [], open, lineEnd);
    expect(fold).not.toBeNull();
    expect(TARIFF.slice(fold!.from - 1, fold!.from)).toBe("{");
    expect(TARIFF.slice(fold!.to, fold!.to + 1)).toBe("}");
  });

  it("folds nothing on a line where no foldable range starts", () => {
    const check = checkTariffSource(rulesSync(), TARIFF, "tariff", CONTEXT);
    const line = at("let linehaul");
    expect(foldAt(check.outline?.folding ?? [], line, TARIFF.indexOf("\n", line))).toBeNull();
  });
});

describe("lint", () => {
  it("places a finding at its range with the module's code", () => {
    const source = 'tariff "x"\ncharge "A" = nope\n';
    const check = tariffCheck(rulesSync(), { source, kind: "tariff", context: CONTEXT });
    const finding = check.diagnostics[0]!;
    const lint = toTariffLint(finding, source);
    expect(lint).toMatchObject({
      from: source.indexOf("nope"),
      to: source.indexOf("nope") + 4,
      severity: "error",
      source: finding.code,
    });
  });

  it("places a library's finding at the line that imports it", () => {
    const lint = toTariffLint(
      {
        code: "type",
        severity: "error",
        message: { message: "Broken" },
        range: { from: 0, to: 3 },
        module: "org.accessorials",
        fixes: [],
        related: [],
      } as never,
      TARIFF,
    );
    const imported = TARIFF.indexOf("org.accessorials");
    expect(lint).toMatchObject({ from: imported, to: imported + "org.accessorials".length });
    expect(lint?.message).toBe("In org.accessorials: Broken");
  });
});

describe("highlighting", () => {
  it("marks only the tokens inside the visible ranges", () => {
    const tokens = tariffTokens(rulesSync(), TARIFF);
    const visible = [{ from: 0, to: TARIFF.indexOf("\n") }];
    const marked: string[] = [];
    visibleTokenDecorations(tokens, visible).between(0, TARIFF.length, (from, to, deco) => {
      marked.push(`${TARIFF.slice(from, to)}:${deco.spec.class}`);
    });
    expect(marked).toEqual([
      "tariff:cm-tariff-keyword cm-syntax-keyword",
      '"Regional per mile":cm-tariff-text cm-syntax-string',
    ]);
  });
});

describe("format", () => {
  it("rewrites the document in the canonical style and keeps the cursor's line", () => {
    const messy = 'tariff "x"\n\n\n\ncharge "A"   =   $5\n';
    const view = mount(messy, []);
    view.dispatch({ selection: { anchor: messy.indexOf("charge") } });
    const result = formatTariffDocument(view, rulesSync());
    expect(result.formatted).not.toBeNull();
    expect(view.state.doc.toString()).toBe(result.formatted);
    expect(view.state.doc.toString()).toContain('charge "A" = $5');
  });

  it("leaves a file with syntax errors as written", () => {
    const broken = 'tariff "x"\ncharge "A" = (\n';
    const view = mount(broken, []);
    const result = formatTariffDocument(view, rulesSync());
    expect(result.formatted).toBeNull();
    expect(result.diagnostics.length).toBeGreaterThan(0);
    expect(view.state.doc.toString()).toBe(broken);
  });
});
