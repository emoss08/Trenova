import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  compareDecimalText,
  describeTariffConstraint,
  outsideTariffConstraint,
  sameTariffValue,
  tariffConstraintViolations,
} from "../tariff-constraint";

const t: TranslateFn = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

const money = (amount: string) => ({ $q: amount, u: "$/mi" });

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
    expect(outsideTariffConstraint(inclusive, money("0.5"))).toBe(false);
    expect(outsideTariffConstraint(inclusive, money("15"))).toBe(false);
    expect(outsideTariffConstraint(halfOpen, money("15"))).toBe(true);
    expect(outsideTariffConstraint(halfOpen, money("14.99"))).toBe(false);
    expect(outsideTariffConstraint(inclusive, money("0.49"))).toBe(true);
  });

  it("checks an open-ended range on its one end", () => {
    const atLeast = { range: { min: { $n: "1" }, max: null, maxInclusive: false } };
    expect(outsideTariffConstraint(atLeast, { $n: "1000000" })).toBe(false);
    expect(outsideTariffConstraint(atLeast, { $n: "0" })).toBe(true);
  });

  it("lets none through and leaves a value in another unit to the server", () => {
    expect(outsideTariffConstraint(inclusive, null)).toBe(false);
    expect(outsideTariffConstraint(inclusive, { $q: "900", u: "$/km" })).toBe(false);
  });

  it("matches a list constraint by value", () => {
    const service = { oneOf: ["Standard", "Guaranteed"] };
    expect(outsideTariffConstraint(service, "Guaranteed")).toBe(false);
    expect(outsideTariffConstraint(service, "Rush")).toBe(true);
    expect(outsideTariffConstraint({ oneOf: [{ $n: "2" }] }, { $n: "2.0" })).toBe(false);
  });

  it("orders dates as moments", () => {
    const window = {
      range: { min: { $date: "2026-01-01" }, max: { $date: "2027-01-01" }, maxInclusive: false },
    };
    expect(outsideTariffConstraint(window, { $date: "2026-06-30" })).toBe(false);
    expect(outsideTariffConstraint(window, { $date: "2027-01-01" })).toBe(true);
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
