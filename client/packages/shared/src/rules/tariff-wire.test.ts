import {
  tariffTypeSchema,
  tariffWireValueSchema,
  type TariffType,
} from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import {
  decodeTariffValue,
  emptyTariffValue,
  encodeTariffValue,
  formatTariffUnit,
  normalizeDecimal,
  parseTariffUnit,
  tariffDate,
  tariffDecimal,
  tariffMoney,
  tariffNumber,
  tariffPercent,
  tariffQuantity,
  tariffTime,
  tariffUnitDimension,
  tariffUnitsOf,
  tariffValueProblems,
  TariffWireError,
} from "./tariff-wire";

describe("times of day", () => {
  it("reads the module's time and the short form, and writes it back with seconds", () => {
    expect(decodeTariffValue({ $time: "17:30:00" })).toEqual({
      kind: "time",
      value: "17:30:00",
      hour: 17,
      minute: 30,
      second: 0,
    });
    const short = decodeTariffValue({ $time: "07:05" });
    expect(short).toEqual({ kind: "time", value: "07:05:00", hour: 7, minute: 5, second: 0 });
    expect(encodeTariffValue(short)).toEqual({ $time: "07:05:00" });
    expect(tariffTime("23:59")).toEqual({ $time: "23:59:00" });
  });

  it("refuses a time outside one day or in another shape", () => {
    expect(() => decodeTariffValue({ $time: "25:00:00" })).toThrow(TariffWireError);
    expect(() => decodeTariffValue({ $time: "7:30" })).toThrow(TariffWireError);
    expect(() => tariffTime("12:60")).toThrow(TariffWireError);
    expect(tariffWireValueSchema.safeParse({ $time: "24:00:00" }).success).toBe(false);
    expect(tariffWireValueSchema.parse({ $time: "17:30" })).toEqual({ $time: "17:30" });
  });

  it("checks a value against a time descriptor, and a time is not a record", () => {
    expect(tariffValueProblems({ kind: "time" }, { $time: "08:00:00" })).toEqual([]);
    expect(tariffValueProblems({ kind: "time" }, { $date: "2026-10-22" })).toEqual([
      { path: "", problem: "not-time" },
    ]);
    expect(tariffValueProblems({ kind: "time" }, "08:00")).toEqual([
      { path: "", problem: "not-time" },
    ]);
    expect(tariffValueProblems({ kind: "optional", inner: { kind: "time" } }, null)).toEqual([]);
    expect(
      tariffValueProblems({ kind: "record", name: null, fields: [] }, { $time: "08:00:00" }),
    ).toEqual([{ path: "", problem: "not-record" }]);
    expect(emptyTariffValue({ kind: "time" })).toBeNull();
  });
});

const breakType: TariffType = {
  kind: "record",
  name: "std.ltl.Break",
  fields: [
    { name: "from", doc: null, type: { kind: "quantity", dimension: "Weight", unit: "lb" } },
    {
      name: "rate",
      doc: null,
      type: { kind: "quantity", dimension: "Money/Weight", unit: "$/cwt" },
    },
    { name: "minimum", doc: null, type: { kind: "optional", inner: { kind: "money" } } },
  ],
};

describe("units", () => {
  it("reads a rate unit into money over a weight", () => {
    const unit = parseTariffUnit("$/cwt");
    expect(unit?.numerator?.symbol).toBe("$");
    expect(unit?.denominator.map((part) => part.symbol)).toEqual(["cwt"]);
    expect(unit && tariffUnitDimension(unit)).toBe("Money/Weight");
    expect(unit && formatTariffUnit(unit)).toBe("$/cwt");
  });

  it("reads a density and a temperature", () => {
    const density = parseTariffUnit("lb/ft3");
    expect(density && tariffUnitDimension(density)).toBe("Weight/Volume");
    expect(parseTariffUnit("°F")?.numerator?.dimension).toBe("Temperature");
  });

  it("folds a count label into the value, as the checker does", () => {
    const dimension = (text: string) => {
      const unit = parseTariffUnit(text);
      return unit && tariffUnitDimension(unit);
    };
    expect(["$/pallet", "$/piece", "$/stop"].map(dimension)).toEqual(["Money", "Money", "Money"]);
    expect(dimension("lb/pallet")).toBe("Weight");
    expect(dimension("pallet")).toBe("Number");
    expect(dimension("$/mi")).toBe("Money/Distance");
  });

  it("refuses a unit the language does not know", () => {
    expect(parseTariffUnit("furlong")).toBeNull();
    expect(parseTariffUnit("$/")).toBeNull();
    expect(parseTariffUnit("")).toBeNull();
  });

  it("lists the units of a dimension for a unit select", () => {
    expect(tariffUnitsOf("Weight")).toEqual(["lb", "cwt", "kg", "t", "ton"]);
    expect(tariffUnitsOf("Money/Distance")).toEqual(["$/mi", "$/km"]);
    expect(tariffUnitsOf("Count")).toEqual(["pallet", "piece", "stop"]);
    expect(tariffUnitsOf("Nothing")).toEqual([]);
  });
});

describe("decimals", () => {
  it("normalises what a person typed into the module's canonical text", () => {
    expect(tariffDecimal(" 002.3500 ")).toBe("2.35");
    expect(tariffDecimal("1200")).toBe("1200");
    expect(tariffDecimal("-0.0")).toBe("0");
    expect(tariffDecimal(18.5)).toBe("18.5");
  });

  it("reads a half-typed value as not filled in", () => {
    expect(tariffDecimal("1.")).toBeNull();
    expect(tariffDecimal("abc")).toBeNull();
    expect(tariffDecimal("1,200")).toBeNull();
    expect(tariffDecimal("1e3")).toBeNull();
    expect(tariffDecimal(Number.NaN)).toBeNull();
    expect(tariffDecimal(null)).toBeNull();
  });

  it("keeps precision past a float's", () => {
    expect(normalizeDecimal("12345678901234567890.123456789")).toBe(
      "12345678901234567890.123456789",
    );
  });
});

describe("encoding", () => {
  it("writes each wire form the reference lists", () => {
    expect(tariffNumber("1200")).toEqual({ $n: "1200" });
    expect(tariffQuantity("2.350", "$/mi")).toEqual({ $q: "2.35", u: "$/mi" });
    expect(tariffMoney("450")).toEqual({ $q: "450", u: "$" });
    expect(tariffPercent("18")).toEqual({ $q: "18", u: "%" });
    expect(tariffDate("2026-10-22")).toEqual({ $date: "2026-10-22" });
  });

  it("round-trips a list of records through decode and encode", () => {
    const wire = [
      { from: { $q: "0", u: "lb" }, rate: { $q: "18.5", u: "$/cwt" }, minimum: null },
      {
        from: { $q: "500", u: "lb" },
        rate: { $q: "16", u: "$/cwt" },
        minimum: { $q: "75", u: "$" },
      },
    ];
    expect(encodeTariffValue(decodeTariffValue(wire))).toEqual(wire);
    expect(tariffWireValueSchema.parse(wire)).toEqual(wire);
  });
});

describe("decoding", () => {
  it("reads a quantity with its unit", () => {
    expect(decodeTariffValue({ $q: "1200.0", u: "lb" })).toMatchObject({
      kind: "quantity",
      amount: "1200",
      unitText: "lb",
    });
  });

  it("reads a date-time's wall clock and offset", () => {
    expect(decodeTariffValue({ $date: "2026-10-22T08:00:00-05:00" })).toEqual({
      kind: "dateTime",
      value: "2026-10-22T08:00:00-05:00",
      date: "2026-10-22",
      hour: 8,
      minute: 0,
      second: 0,
      offset: "-05:00",
    });
    expect(decodeTariffValue({ $date: "2026-10-22T08:00Z" })).toMatchObject({ offset: "+00:00" });
  });

  it("names where a malformed value is", () => {
    expect(() => decodeTariffValue({ rows: [{ $q: "x", u: "lb" }] })).toThrow(TariffWireError);
    try {
      decodeTariffValue({ rows: [{ $q: "x", u: "lb" }] });
    } catch (error) {
      expect((error as TariffWireError).path).toBe("rows[0]");
    }
    expect(() => decodeTariffValue({ $q: "1", u: "parsec" })).toThrow("parsec is not a unit");
    expect(() => decodeTariffValue({ $date: "Tuesday" })).toThrow(TariffWireError);
    expect(() => decodeTariffValue({ $tag: 1 })).toThrow(TariffWireError);
  });
});

describe("tariffValueProblems", () => {
  it("accepts a weight-break table that fits std.ltl.Break", () => {
    const wire = [{ from: { $q: "0", u: "lb" }, rate: { $q: "18.5", u: "$/cwt" } }];
    expect(tariffValueProblems({ kind: "list", item: breakType }, wire)).toEqual([]);
  });

  it("accepts any unit of the declared dimension", () => {
    const type: TariffType = { kind: "quantity", dimension: "Weight", unit: "lb" };
    expect(tariffValueProblems(type, { $q: "2", u: "cwt" })).toEqual([]);
    expect(tariffValueProblems(type, { $q: "2", u: "mi" })).toEqual([
      { path: "", problem: "wrong-unit", detail: "Weight" },
    ]);
  });

  it("finds a rate per pound where the type is per mile, a missing field and an extra one", () => {
    const wire = [{ from: { $q: "0", u: "mi" }, cost: { $q: "1", u: "$" } }];
    expect(tariffValueProblems({ kind: "list", item: breakType }, wire)).toEqual([
      { path: "[0].from", problem: "wrong-unit", detail: "Weight" },
      { path: "[0].rate", problem: "missing" },
      { path: "[0]", problem: "unknown-field", detail: "cost" },
    ]);
  });

  it("takes money per pallet, piece or stop as money, and no other rate", () => {
    for (const u of ["$", "$/pallet", "$/piece", "$/stop"]) {
      expect(tariffValueProblems({ kind: "money" }, { $q: "45", u })).toEqual([]);
    }
    for (const u of ["$/mi", "lb", "%"]) {
      expect(tariffValueProblems({ kind: "money" }, { $q: "45", u })).toEqual([
        { path: "", problem: "not-money" },
      ]);
    }
  });

  it("takes none only for an optional, and whole numbers only for an Int", () => {
    expect(tariffValueProblems({ kind: "money" }, null)).toEqual([
      { path: "", problem: "not-money" },
    ]);
    expect(tariffValueProblems({ kind: "optional", inner: { kind: "money" } }, null)).toEqual([]);
    expect(tariffValueProblems({ kind: "int" }, { $n: "2.5" })).toEqual([
      { path: "", problem: "not-int" },
    ]);
    expect(tariffValueProblems({ kind: "int" }, { $n: "2.0" })).toEqual([]);
  });

  it("checks enum variants", () => {
    const service: TariffType = {
      kind: "enum",
      name: "Service",
      variants: ["Standard", "Expedited"],
      ordered: false,
    };
    expect(tariffValueProblems(service, "Expedited")).toEqual([]);
    expect(tariffValueProblems(service, "Overnight")).toEqual([
      { path: "", problem: "not-variant", detail: "Standard, Expedited" },
    ]);
  });
});

describe("emptyTariffValue", () => {
  it("starts a new weight-break row with zeros in the declared units", () => {
    expect(emptyTariffValue(breakType)).toEqual({
      from: { $q: "0", u: "lb" },
      rate: { $q: "0", u: "$/cwt" },
      minimum: null,
    });
  });
});

describe("tariffTypeSchema", () => {
  it("reads the descriptors tariff.check writes", () => {
    expect(
      tariffTypeSchema.parse({
        kind: "list",
        item: {
          kind: "record",
          name: "Break",
          fields: [{ name: "from", type: { kind: "quantity", dimension: "Weight", unit: "lb" } }],
        },
      }),
    ).toEqual({
      kind: "list",
      item: {
        kind: "record",
        name: "Break",
        fields: [
          { name: "from", doc: null, type: { kind: "quantity", dimension: "Weight", unit: "lb" } },
        ],
      },
    });
  });

  it("reads a time of day and whether an enum's variants are ordered", () => {
    expect(tariffTypeSchema.parse({ kind: "optional", inner: { kind: "time" } })).toEqual({
      kind: "optional",
      inner: { kind: "time" },
    });
    expect(
      tariffTypeSchema.parse({
        kind: "enum",
        name: "Weekday",
        ordered: true,
        variants: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
      }),
    ).toMatchObject({ kind: "enum", name: "Weekday", ordered: true });
    expect(
      tariffTypeSchema.parse({ kind: "enum", name: "StopKind", variants: ["Pickup", "Delivery"] }),
    ).toEqual({ kind: "enum", name: "StopKind", variants: ["Pickup", "Delivery"], ordered: false });
  });
});
