import { describe, expect, it } from "vitest";
import { cardDay, cardMoment, money, neighborsIn } from "../record-card/record-card-values";

describe("neighborsIn", () => {
  const order = ["ft_1", "ft_2", "ft_3"];

  it("has no row before the first", () => {
    expect(neighborsIn(order, "ft_1")).toEqual({
      prevId: null,
      nextId: "ft_2",
      position: 1,
      total: 3,
    });
  });

  it("has a row either side of one in the middle", () => {
    expect(neighborsIn(order, "ft_2")).toEqual({
      prevId: "ft_1",
      nextId: "ft_3",
      position: 2,
      total: 3,
    });
  });

  it("has no row after the last", () => {
    expect(neighborsIn(order, "ft_3")).toEqual({
      prevId: "ft_2",
      nextId: null,
      position: 3,
      total: 3,
    });
  });

  it("places a record the list does not hold nowhere, and still counts the list", () => {
    expect(neighborsIn(order, "ft_9")).toEqual({
      prevId: null,
      nextId: null,
      position: 0,
      total: 3,
    });
  });

  it("steps nowhere in a list of one", () => {
    expect(neighborsIn(["ft_1"], "ft_1")).toEqual({
      prevId: null,
      nextId: null,
      position: 1,
      total: 1,
    });
  });
});

describe("money", () => {
  it("writes a dash for an amount that is not there", () => {
    expect(money(null)).toBe("—");
    expect(money(undefined)).toBe("—");
    expect(money("")).toBe("—");
    expect(money("not a figure")).toBe("—");
    expect(money(Number.NaN)).toBe("—");
  });

  it("writes a figure or a decimal string as dollars by default", () => {
    const dollars = (figure: number) =>
      new Intl.NumberFormat(undefined, {
        style: "currency",
        currency: "USD",
        minimumFractionDigits: 2,
      }).format(figure);

    expect(money(1250)).toBe(dollars(1250));
    expect(money("1121.36")).toBe(dollars(1121.36));
    expect(money(0)).toBe(dollars(0));
  });

  it("writes the currency it is given", () => {
    expect(money(10, "EUR")).toBe(
      new Intl.NumberFormat(undefined, {
        style: "currency",
        currency: "EUR",
        minimumFractionDigits: 2,
      }).format(10),
    );
  });
});

describe("card dates", () => {
  it("say nothing for a moment that is not set", () => {
    expect(cardMoment(0)).toBe("");
    expect(cardMoment(null)).toBe("");
    expect(cardMoment(undefined)).toBe("");
    expect(cardDay(0)).toBe("");
    expect(cardDay(null)).toBe("");
    expect(cardDay(undefined)).toBe("");
  });

  it("say the day of a moment that is set", () => {
    expect(cardDay(1_790_000_000)).not.toBe("");
    expect(cardMoment(1_790_000_000)).not.toBe("");
  });
});
