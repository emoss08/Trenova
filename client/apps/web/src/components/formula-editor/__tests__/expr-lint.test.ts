import { forceLinting, forEachDiagnostic, type Diagnostic } from "@codemirror/lint";
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { requireCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { createExprLinter, lintFindings, toCodeMirrorDiagnostic } from "../expr-lint";
import { checkExpression } from "../formula-check";
import { knownFor } from "./formula-fixtures";

const rules = rulesSync();
const known = knownFor([{ name: "fuelPct", type: "Number", description: "", required: false }]);

function findings(source: string) {
  return lintFindings(checkExpression(rules, source, known).diagnostics);
}

beforeAll(() => {
  requireCatalog("core", "web");
});

afterEach(async () => {
  await setLocale("en");
});

describe("lintFindings", () => {
  it("underlines nothing in a formula that saves", () => {
    expect(findings("round(baseRate * totalDistance * (1 + fuelPct / 100), 2)")).toEqual([]);
  });

  it("underlines nothing in an empty editor, though the check says it is empty", () => {
    expect(checkExpression(rules, "   ", known).diagnostics.map((d) => d.code)).toEqual(["empty"]);
    expect(findings("   ")).toEqual([]);
  });

  it("keeps every other finding, in the module's order", () => {
    expect(findings("let a = 1;\nbaseRaet * 2").map((d) => [d.code, d.from])).toEqual([
      ["let-unused", 4],
      ["unknown-variable", 11],
    ]);
  });
});

describe("toCodeMirrorDiagnostic", () => {
  it("draws the translated message and offers the module's fixes as actions", async () => {
    const [unknown] = findings("baseRaet * 2");
    const declare = vi.fn();

    const drawn = toCodeMirrorDiagnostic(unknown, declare);
    expect(drawn).toMatchObject({
      from: 0,
      to: 8,
      severity: "error",
      source: "unknown-variable",
      message:
        "`baseRaet` isn't a shipment field or one of this template's inputs. Did you mean `baseRate`?",
    });
    expect(drawn.actions?.map((action) => action.name)).toEqual([
      "Change to baseRate",
      "Declare baseRaet as a Number input",
    ]);

    drawn.actions?.[1]?.apply(null as never, 0, 8);
    expect(declare).toHaveBeenCalledWith("baseRaet", "Number");
  });

  it("reads in the language on screen", async () => {
    const [unknown] = findings("weekendFee * 2");
    await setLocale("es");
    expect(toCodeMirrorDiagnostic(unknown).message).toBe(
      "`weekendFee` no es un campo del envío ni una entrada de esta plantilla. Decláralo como entrada o elige un campo de la referencia.",
    );
  });

  it("leaves out a declare fix when nothing can declare an input", () => {
    const [unknown] = findings("weekendFee * 2");
    expect(toCodeMirrorDiagnostic(unknown).actions ?? []).toEqual([]);
  });
});

describe("createExprLinter", () => {
  it("lints the editor's document with the rules module and applies a fix in place", async () => {
    const view = new EditorView({
      state: EditorState.create({
        doc: "baseRaet * 2",
        extensions: [
          createExprLinter(
            () => rules,
            () => known,
          ),
        ],
      }),
    });
    forceLinting(view);
    await vi.waitFor(() => {
      const drawn: Diagnostic[] = [];
      forEachDiagnostic(view.state, (diagnostic) => drawn.push(diagnostic));
      expect(drawn.map((diagnostic) => diagnostic.source)).toEqual(["unknown-variable"]);
    });

    const drawn: Diagnostic[] = [];
    forEachDiagnostic(view.state, (diagnostic) => drawn.push(diagnostic));
    drawn[0]?.actions?.[0]?.apply(view, 0, 8);
    expect(view.state.doc.toString()).toBe("baseRate * 2");
    view.destroy();
  });
});
