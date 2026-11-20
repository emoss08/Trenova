import { translate } from "@trenova/shared/i18n/runtime";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { lineTemperatures, loadTemperature, temperatureLabel, temperatureOf } from "../temperature";

const t = translate as TranslateFn;

describe("temperatureLabel", () => {
  it("says a range, one bound or nothing", () => {
    expect(temperatureLabel({ min: 34, max: 38 }, t)).toBe("34–38°F");
    expect(temperatureLabel({ min: 34, max: null }, t)).toBe("≥ 34°F");
    expect(temperatureLabel({ min: null, max: -5 }, t)).toBe("≤ -5°F");
    expect(temperatureLabel({ min: 35, max: 35 }, t)).toBe("35°F");
    expect(temperatureLabel({ min: null, max: null }, t)).toBe("");
  });
});

describe("temperatureOf", () => {
  it("reads an absent bound and a null one alike", () => {
    expect(temperatureOf({ temperatureMin: 0 })).toEqual({ min: 0, max: null });
    expect(temperatureOf({ temperatureMin: null, temperatureMax: 10 })).toEqual({
      min: null,
      max: 10,
    });
    expect(temperatureOf(undefined)).toEqual({ min: null, max: null });
  });
});

describe("lineTemperatures", () => {
  it("takes the ranges of live stops and of every commodity line, skipping unset ones", () => {
    const ranges = lineTemperatures(
      [
        {
          status: "New",
          stops: [
            { status: "New" },
            { status: "New", temperatureMin: -10, temperatureMax: 0 },
            { status: "Canceled", temperatureMin: 60, temperatureMax: 70 },
          ],
        },
        { status: "Canceled", stops: [{ status: "New", temperatureMin: 80 }] },
      ],
      [{ temperatureMin: 34, temperatureMax: 38 }, {}],
    );
    expect(ranges).toEqual([
      { min: -10, max: 0 },
      { min: 34, max: 38 },
    ]);
  });

  it("is empty for a shipment with no moves or lines", () => {
    expect(lineTemperatures(undefined, null)).toEqual([]);
  });
});

describe("loadTemperature", () => {
  it("is null when no line sets a range", () => {
    expect(loadTemperature([])).toBeNull();
  });

  it("keeps the range every line shares", () => {
    expect(
      loadTemperature([
        { min: 28, max: 38 },
        { min: 32, max: 40 },
      ]),
    ).toEqual({ min: 32, max: 38 });
    expect(
      loadTemperature([
        { min: 28, max: 38 },
        { min: null, max: 34 },
      ]),
    ).toEqual({ min: 28, max: 34 });
  });

  it("spans the coldest to the warmest bound when the lines share nothing", () => {
    expect(
      loadTemperature([
        { min: -10, max: 0 },
        { min: 34, max: 38 },
      ]),
    ).toEqual({ min: -10, max: 38 });
    expect(
      loadTemperature([
        { min: null, max: 0 },
        { min: 34, max: null },
      ]),
    ).toEqual({ min: 0, max: 34 });
  });
});
