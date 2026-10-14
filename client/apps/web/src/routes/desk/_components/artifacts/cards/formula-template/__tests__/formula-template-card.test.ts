import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { VariableDefinition } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import {
  formulaStage,
  guardrailNote,
  isEditableVariable,
  rootOf,
  roundingLabel,
  roundingNote,
  sampleLoadSummary,
  tokenizeFormula,
  usageLabel,
  variableDraft,
  variableDrafts,
  variableOverrides,
  variableValue,
  type FormulaToken,
} from "../formula-template-card";
import { money } from "../../../record-card/record-card-values";
import { describedShipment } from "@/components/formula-editor/__tests__/described-shipment";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

const known = buildKnownIdentifiers(describedShipment, []);

function tokens(expression: string, variables: string[] = []): FormulaToken[] {
  return tokenizeFormula(expression, known, new Set(variables));
}

function kinds(expression: string, variables: string[] = []): Array<[string, string]> {
  return tokens(expression, variables)
    .filter((token) => token.kind !== "plain")
    .map((token) => [token.text, token.kind]);
}

function variable(extra: Partial<VariableDefinition> & Pick<VariableDefinition, "name">) {
  return {
    type: "Number",
    description: "",
    required: false,
    ...extra,
  } as VariableDefinition;
}

describe("tokenizeFormula", () => {
  it.each([
    "baseRate * totalDistance",
    "max(minimumCharge, baseRate * totalDistance)\n  + fuelSurcharge",
    "// linehaul\nbaseRate * totalDistance /* per mile */ + stopFee",
    '"unterminated string',
    "/* never closed",
    'customer.name == "Acme" ? 1 : 0',
    "a?.b ?? fallback",
    "2 ** 3",
    "1..5",
    "1_000.5 + .5 + 1e3 + 0x1F",
    "x ; y @ z",
  ])("gives back %j exactly when its pieces are joined", (expression) => {
    expect(
      tokens(expression)
        .map((token) => token.text)
        .join(""),
    ).toBe(expression);
  });

  it("marks the template's own variables apart from the shipment's fields", () => {
    expect(kinds("baseRate * totalDistance", ["baseRate"])).toEqual([
      ["baseRate", "var"],
      ["*", "op"],
      ["totalDistance", "field"],
    ]);
  });

  it("marks a path through a template variable as the variable", () => {
    expect(kinds("rates.reefer + 1", ["rates"])).toEqual([
      ["rates.reefer", "var"],
      ["+", "op"],
      ["1", "num"],
    ]);
  });

  it("marks calls as functions, known or not, and an unknown name as neither", () => {
    expect(kinds("round(mystery(qqq))")).toEqual([
      ["round", "fn"],
      ["mystery", "fn"],
      ["qqq", "ident"],
    ]);
    expect(kinds("round (1)")).toEqual([
      ["round", "fn"],
      ["1", "num"],
    ]);
  });

  it("marks keywords and operator words", () => {
    expect(kinds('if true and qqq contains "x"')).toEqual([
      ["if", "kw"],
      ["true", "kw"],
      ["and", "kw"],
      ["qqq", "ident"],
      ["contains", "op"],
      ['"x"', "str"],
    ]);
  });

  it("reads numbers, strings, comments and operators whole", () => {
    expect(kinds("1_000.5 .5 1e3 'a' `b` ** ?? .. // done")).toEqual([
      ["1_000.5", "num"],
      [".5", "num"],
      ["1e3", "num"],
      ["'a'", "str"],
      ["`b`", "str"],
      ["**", "op"],
      ["??", "op"],
      ["..", "op"],
      ["// done", "comment"],
    ]);
  });

  it("keeps a run of spacing and stray characters as one plain piece", () => {
    expect(tokens("qqq ; zzz")).toEqual([
      { text: "qqq", kind: "ident" },
      { text: " ; ", kind: "plain" },
      { text: "zzz", kind: "ident" },
    ]);
    expect(tokens("qqq  \n\t zzz").map((token) => token.kind)).toEqual(["ident", "plain", "ident"]);
  });
});

describe("rootOf", () => {
  it("names the variable a path starts from", () => {
    expect(rootOf("baseRate")).toBe("baseRate");
    expect(rootOf("customer.name")).toBe("customer");
    expect(rootOf("a?.b")).toBe("a");
  });
});

describe("formulaStage", () => {
  it("reads every status", () => {
    expect(formulaStage("Active")).toBe("active");
    expect(formulaStage("InReview")).toBe("review");
    expect(formulaStage("Inactive")).toBe("inactive");
    expect(formulaStage("Draft")).toBe("draft");
  });
});

describe("variableValue", () => {
  it("reads a number, commas and all, and refuses one that is not", () => {
    expect(variableValue("Number", "1,250.5")).toBe(1250.5);
    expect(variableValue("Number", " 3 ")).toBe(3);
    expect(variableValue("Number", "abc")).toBeUndefined();
    expect(variableValue("Number", "   ")).toBeUndefined();
  });

  it("reads yes or no and nothing else", () => {
    expect(variableValue("Boolean", "true")).toBe(true);
    expect(variableValue("Boolean", "false")).toBe(false);
    expect(variableValue("Boolean", "yes")).toBeUndefined();
  });

  it("reads any as a number when it is one and as text when not", () => {
    expect(variableValue("Any", "42")).toBe(42);
    expect(variableValue("Any", "reefer")).toBe("reefer");
    expect(variableValue("Any", "")).toBeUndefined();
  });

  it("trims text", () => {
    expect(variableValue("String", "  DV  ")).toBe("DV");
  });
});

describe("variableOverrides", () => {
  it("keeps the values a person set that the engine can take", () => {
    const definitions = [
      variable({ name: "baseRate" }),
      variable({ name: "teamService", type: "Boolean" }),
      variable({ name: "stops", type: "Array" }),
      variable({ name: "rates", type: "Object" }),
      variable({ name: "untouched" }),
      variable({ name: "broken" }),
    ];

    expect(
      variableOverrides(definitions, {
        baseRate: "2.95",
        teamService: "true",
        stops: "[1,2]",
        rates: '{"a":1}',
        broken: "two",
      }),
    ).toEqual({ baseRate: 2.95, teamService: true });
  });
});

describe("variableDraft", () => {
  it("shows a default the way its field holds it", () => {
    expect(variableDraft(variable({ name: "a", defaultValue: null }))).toBe("");
    expect(variableDraft(variable({ name: "a", defaultValue: undefined }))).toBe("");
    expect(variableDraft(variable({ name: "a", defaultValue: true }))).toBe("true");
    expect(variableDraft(variable({ name: "a", defaultValue: false }))).toBe("false");
    expect(variableDraft(variable({ name: "a", defaultValue: 2.85 }))).toBe("2.85");
    expect(variableDraft(variable({ name: "a", defaultValue: "DV" }))).toBe("DV");
    expect(variableDraft(variable({ name: "a", defaultValue: { band: 1 } }))).toBe('{"band":1}');
  });
});

describe("isEditableVariable", () => {
  it("takes plain values in a field and leaves lists, records and dates to the studio", () => {
    expect(isEditableVariable("Number")).toBe(true);
    expect(isEditableVariable("String")).toBe(true);
    expect(isEditableVariable("Boolean")).toBe(true);
    expect(isEditableVariable("Any")).toBe(true);
    expect(isEditableVariable("Date")).toBe(false);
    expect(isEditableVariable("Array")).toBe(false);
    expect(isEditableVariable("Object")).toBe(false);
  });
});

describe("guardrailNote", () => {
  it("says nothing when no limit held the price", () => {
    expect(guardrailNote(null, t)).toBe("");
    expect(guardrailNote(undefined, t)).toBe("");
    expect(guardrailNote({ applied: false, rawAmount: 120.4 }, t)).toBe("");
  });

  it("says which limit held the price and from what", () => {
    expect(guardrailNote({ applied: true, bound: "max", rawAmount: 6200 }, t)).toBe(
      `Held to the maximum, down from ${money(6200)}`,
    );
    expect(guardrailNote({ applied: true, bound: "min", rawAmount: 120.4 }, t)).toBe(
      `Raised to the minimum, up from ${money(120.4)}`,
    );
  });
});

describe("usageLabel", () => {
  it("counts each kind of use the way a person says it", () => {
    expect(usageLabel({ type: "shipment", count: 1 }, t)).toBe("1 shipment");
    expect(usageLabel({ type: "shipment", count: 3 }, t)).toBe("3 shipments");
    expect(usageLabel({ type: "rate_matrix", count: 2 }, t)).toBe("2 rate matrices");
    expect(usageLabel({ type: "rate_agreement_rule", count: 1 }, t)).toBe("1 agreement rule");
    expect(usageLabel({ type: "rate_agreement_accessorial", count: 2 }, t)).toBe(
      "2 agreement accessorials",
    );
    expect(usageLabel({ type: "invoice_line", count: 4 }, t)).toBe("4 other uses");
  });
});

describe("roundingLabel", () => {
  it("says how and to how many places", () => {
    expect(roundingLabel("None", 2, t)).toBe("Not rounded");
    expect(roundingLabel("HalfUp", 2, t)).toBe("Half up, 2 places");
    expect(roundingLabel("HalfEven", 1, t)).toBe("Half to even, 1 place");
    expect(roundingLabel("Up", 0, t)).toBe("Up, 0 places");
    expect(roundingLabel("Down", 3, t)).toBe("Down, 3 places");
  });
});

describe("variableDrafts", () => {
  it("fills every variable's field with its default", () => {
    expect(
      variableDrafts([
        variable({ name: "baseRate", defaultValue: 2.85 }),
        variable({ name: "teamService", type: "Boolean", defaultValue: false }),
        variable({ name: "note", type: "String" }),
      ]),
    ).toEqual({ baseRate: "2.85", teamService: "false", note: "" });
  });
});

describe("roundingNote", () => {
  it("gives the figure before rounding to the places the engine kept", () => {
    expect(roundingNote(1499.98765, t)).toBe("Rounded from 1,499.98765");
    expect(roundingNote(1500, t)).toBe("Rounded from 1,500.00");
  });
});

describe("sampleLoadSummary", () => {
  it("says where the sample load runs, how far and how heavy", () => {
    expect(
      sampleLoadSummary(
        {
          "origin.city": "Atlanta",
          "origin.state": "GA",
          "destination.city": "Miami",
          "destination.state": "FL",
          totalDistance: 500,
          totalWeight: 10000,
        },
        t,
      ),
    ).toEqual(["Atlanta, GA to Miami, FL", "500 mi", "10,000 lb"]);
  });

  it("leaves out what the values do not hold", () => {
    expect(sampleLoadSummary({ "origin.city": "Atlanta", totalDistance: 0 }, t)).toEqual([]);
    expect(sampleLoadSummary({ totalWeight: "heavy" }, t)).toEqual([]);
  });
});
