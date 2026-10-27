import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { TariffType } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import {
  compareCodePoints,
  compareDecimalText,
  describeTariffConstraint,
  orderedEnumChoices,
  outsideTariffConstraint,
  sameTariffValue,
  tariffConstraintViolations,
} from "../tariff-constraint";

const t: TranslateFn = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

const money = (amount: string) => ({ $q: amount, u: "$/mi" });
const rate: TariffType = { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" };
const text: TariffType = { kind: "text" };

// The descriptor tariff.check writes for `FreightClass` (testdata/tariff/check/ordered-enums.json).
const freightClass: TariffType = {
  kind: "enum",
  name: "FreightClass",
  ordered: true,
  variants: [
    "Class50",
    "Class55",
    "Class60",
    "Class65",
    "Class70",
    "Class77_5",
    "Class85",
    "Class92_5",
    "Class100",
    "Class110",
    "Class125",
    "Class150",
    "Class175",
    "Class200",
    "Class250",
    "Class300",
    "Class400",
    "Class500",
  ],
};

describe("compareDecimalText", () => {
  it("orders decimals exactly, whatever their scale", () => {
    expect(compareDecimalText("0.50", "0.5")).toBe(0);
    expect(compareDecimalText("0.1", "0.09999999999999999999")).toBe(1);
    expect(compareDecimalText("-2", "1.5")).toBe(-1);
    expect(compareDecimalText("abc", "1")).toBeNull();
  });
});

describe("outsideTariffConstraint", () => {
  const inclusive = {
    range: { min: money("0.5"), max: money("15"), maxInclusive: true },
  };
  const halfOpen = {
    range: { min: money("0.5"), max: money("15"), maxInclusive: false },
  };

  it("keeps the minimum inclusive and the maximum inclusive only when asked", () => {
    expect(outsideTariffConstraint(inclusive, money("0.5"), rate)).toBe(false);
    expect(outsideTariffConstraint(inclusive, money("15"), rate)).toBe(false);
    expect(outsideTariffConstraint(halfOpen, money("15"), rate)).toBe(true);
    expect(outsideTariffConstraint(halfOpen, money("14.99"), rate)).toBe(false);
    expect(outsideTariffConstraint(inclusive, money("0.49"), rate)).toBe(true);
  });

  it("checks an open-ended range on its one end", () => {
    const atLeast = { range: { min: { $n: "1" }, max: null, maxInclusive: false } };
    expect(outsideTariffConstraint(atLeast, { $n: "1000000" }, { kind: "int" })).toBe(false);
    expect(outsideTariffConstraint(atLeast, { $n: "0" }, { kind: "int" })).toBe(true);
  });

  it("lets none through and leaves a value in another unit to the server", () => {
    expect(outsideTariffConstraint(inclusive, null, rate)).toBe(false);
    expect(outsideTariffConstraint(inclusive, { $q: "900", u: "$/km" }, rate)).toBe(false);
  });

  it("matches a list constraint by value", () => {
    const service = { oneOf: ["Standard", "Guaranteed"] };
    expect(outsideTariffConstraint(service, "Guaranteed", text)).toBe(false);
    expect(outsideTariffConstraint(service, "Rush", text)).toBe(true);
    expect(
      outsideTariffConstraint({ oneOf: [{ $n: "2" }] }, { $n: "2.0" }, { kind: "number" }),
    ).toBe(false);
    expect(
      outsideTariffConstraint(
        { oneOf: [{ $time: "08:00:00" }] },
        { $time: "08:00" },
        {
          kind: "time",
        },
      ),
    ).toBe(false);
  });

  it("orders dates as moments", () => {
    const window = {
      range: { min: { $date: "2026-01-01" }, max: { $date: "2027-01-01" }, maxInclusive: false },
    };
    expect(outsideTariffConstraint(window, { $date: "2026-06-30" }, { kind: "date" })).toBe(false);
    expect(outsideTariffConstraint(window, { $date: "2027-01-01" }, { kind: "date" })).toBe(true);
  });

  // `input open: Time = @08:00 where @06:00..@17:00` (testdata/tariff/check/time.json).
  it("orders times of day, the short form and the module's alike", () => {
    const hours = {
      range: { min: { $time: "06:00:00" }, max: { $time: "17:00:00" }, maxInclusive: false },
    };
    const time: TariffType = { kind: "time" };
    expect(outsideTariffConstraint(hours, { $time: "06:00:00" }, time)).toBe(false);
    expect(outsideTariffConstraint(hours, { $time: "16:59:59" }, time)).toBe(false);
    expect(outsideTariffConstraint(hours, { $time: "17:00" }, time)).toBe(true);
    expect(outsideTariffConstraint(hours, { $time: "05:59" }, time)).toBe(true);
  });

  // `input zip: Text = "30301" where "30000"..="39999"` (testdata/tariff/check/text-ranges.json).
  it("orders text by code point, never by locale or by number", () => {
    const zips = { range: { min: "30000", max: "39999", maxInclusive: true } };
    expect(outsideTariffConstraint(zips, "30301", text)).toBe(false);
    expect(outsideTariffConstraint(zips, "39999", text)).toBe(false);
    expect(outsideTariffConstraint(zips, "40000", text)).toBe(true);
    expect(outsideTariffConstraint(zips, "3", text)).toBe(true);
    expect(outsideTariffConstraint(zips, "300001", text)).toBe(false);
    expect(outsideTariffConstraint(zips, "3999999", text)).toBe(true);
    const letters = { range: { min: "a", max: "z", maxInclusive: true } };
    expect(outsideTariffConstraint(letters, "B", text)).toBe(true);
  });

  // `input cls: FreightClass = Class70 where Class50..=Class100` (ordered-enums.json).
  it("orders an ordered enum's variants by their place, not their spelling", () => {
    const classes = { range: { min: "Class50", max: "Class100", maxInclusive: true } };
    expect(outsideTariffConstraint(classes, "Class70", freightClass)).toBe(false);
    expect(outsideTariffConstraint(classes, "Class100", freightClass)).toBe(false);
    expect(outsideTariffConstraint(classes, "Class110", freightClass)).toBe(true);
    const below = { range: { min: null, max: "Class100", maxInclusive: false } };
    expect(outsideTariffConstraint(below, "Class92_5", freightClass)).toBe(false);
    expect(outsideTariffConstraint(below, "Class100", freightClass)).toBe(true);
  });

  it("leaves an unordered enum's range and an unknown variant to the server", () => {
    const stops: TariffType = {
      kind: "enum",
      name: "StopKind",
      ordered: false,
      variants: ["Pickup", "Delivery"],
    };
    const range = { range: { min: "Pickup", max: "Pickup", maxInclusive: true } };
    expect(outsideTariffConstraint(range, "Delivery", stops)).toBe(false);
    const classes = { range: { min: "Class50", max: "Class100", maxInclusive: true } };
    expect(outsideTariffConstraint(classes, "Class1000", freightClass)).toBe(false);
  });
});

describe("compareCodePoints", () => {
  it("puts a character past the Basic Multilingual Plane after every one inside it", () => {
    const emoji = String.fromCodePoint(0x1f600);
    const tilde = String.fromCodePoint(0xff5e);
    expect(emoji < tilde).toBe(true);
    expect(compareCodePoints(emoji, tilde)).toBe(1);
    expect(compareCodePoints(tilde, emoji)).toBe(-1);
    expect(compareCodePoints("ab", "abc")).toBe(-1);
    expect(compareCodePoints("abc", "abc")).toBe(0);
    expect(compareCodePoints("b", `a${emoji}`)).toBe(1);
  });
});

describe("orderedEnumChoices", () => {
  it("offers the variants an ordered enum's range spans, in order", () => {
    expect(
      orderedEnumChoices(
        { range: { min: "Class50", max: "Class100", maxInclusive: true } },
        freightClass,
      ),
    ).toEqual([
      "Class50",
      "Class55",
      "Class60",
      "Class65",
      "Class70",
      "Class77_5",
      "Class85",
      "Class92_5",
      "Class100",
    ]);
    expect(
      orderedEnumChoices(
        { range: { min: "Class400", max: null, maxInclusive: false } },
        { kind: "optional", inner: freightClass },
      ),
    ).toEqual(["Class400", "Class500"]);
  });

  it("offers nothing for text, a list constraint or no constraint", () => {
    expect(orderedEnumChoices({ range: { min: "a", max: "b", maxInclusive: true } }, text)).toBe(
      null,
    );
    expect(orderedEnumChoices({ oneOf: ["Class50"] }, freightClass)).toBeNull();
    expect(orderedEnumChoices(null, freightClass)).toBeNull();
  });
});

describe("tariffConstraintViolations", () => {
  it("names each list item outside the bounds", () => {
    const constraint = { oneOf: ["A", "B"] };
    expect(
      tariffConstraintViolations(constraint, { kind: "list", item: { kind: "text" } }, [
        "A",
        "C",
        "B",
        "D",
      ]),
    ).toEqual([1, 3]);
  });

  it("answers -1 for a single value outside the bounds", () => {
    expect(tariffConstraintViolations({ oneOf: ["A"] }, { kind: "text" }, "B")).toEqual([-1]);
    expect(tariffConstraintViolations({ oneOf: ["A"] }, { kind: "text" }, undefined)).toEqual([]);
  });

  it("orders a list input's items by the item type", () => {
    const classes = { range: { min: "Class50", max: "Class100", maxInclusive: true } };
    expect(
      tariffConstraintViolations(
        classes,
        { kind: "optional", inner: { kind: "list", item: freightClass } },
        ["Class55", "Class150", "Class100", "Class500"],
      ),
    ).toEqual([1, 3]);
  });
});

describe("sameTariffValue", () => {
  it("compares records and lists deeply", () => {
    expect(sameTariffValue({ a: [{ $n: "1" }] }, { a: [{ $n: "1.00" }] })).toBe(true);
    expect(sameTariffValue({ a: [{ $n: "1" }] }, { a: [{ $n: "2" }] })).toBe(false);
  });
});

describe("describeTariffConstraint", () => {
  it("writes each form of the bounds in words", () => {
    expect(
      describeTariffConstraint(
        { range: { min: money("0.5"), max: money("15"), maxInclusive: true } },
        t,
      ),
    ).toBe("From $0.50/mi to $15.00/mi");
    expect(
      describeTariffConstraint(
        { range: { min: { $n: "0" }, max: { $n: "10" }, maxInclusive: false } },
        t,
      ),
    ).toBe("At least 0 and below 10");
    expect(
      describeTariffConstraint({ range: { min: null, max: { $n: "10" }, maxInclusive: true } }, t),
    ).toBe("At most 10");
    expect(
      describeTariffConstraint({ range: { min: null, max: { $n: "10" }, maxInclusive: false } }, t),
    ).toBe("Below 10");
    expect(
      describeTariffConstraint({ range: { min: { $n: "1" }, max: null, maxInclusive: false } }, t),
    ).toBe("At least 1");
    expect(describeTariffConstraint({ oneOf: ["Standard", "Guaranteed"] }, t)).toBe(
      "One of Standard, Guaranteed",
    );
  });

  it("says a list input's bounds hold for each value", () => {
    expect(describeTariffConstraint({ oneOf: ["A", "B"] }, t, { each: true })).toBe(
      "Each one of A, B",
    );
  });

  it("shows money in the rating currency", () => {
    expect(
      describeTariffConstraint(
        { range: { min: { $q: "5", u: "$" }, max: null, maxInclusive: false } },
        t,
        { currency: "CAD" },
      ),
    ).toBe("At least CA$5.00");
  });
});
