import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { moveProblems, problemText, problemTextInMove } from "../move-problems";

// Error trees shaped the way react-hook-form holds them under `errors.moves[i]`: a field error
// is { type, message, ref }; a field array is an array (holes where an item has no error) that
// may carry its own error under `root`; an array-level error with no item errors is a plain
// field error on the array.
const required = (message: string) => ({ type: "too_small", message, ref: {} });

describe("moveProblems", () => {
  it("places a stop's field error on that stop and field, with its form path", () => {
    const errors = { stops: [undefined, { locationId: required("Location is required") }] };

    expect(moveProblems(2, errors)).toEqual([
      {
        stopIndex: 1,
        field: "locationId",
        path: "moves.2.stops.1.locationId",
        message: "Location is required",
      },
    ]);
  });

  it("keeps every stop's problems apart, in stop order, when several stops have them", () => {
    const errors = {
      stops: [
        {
          locationId: required("Location is required"),
          weight: required("Weight must be a whole number"),
        },
        undefined,
        { scheduledWindowEnd: required("The window must end after it starts") },
      ],
    };

    expect(moveProblems(0, errors).map(({ stopIndex, field }) => ({ stopIndex, field }))).toEqual([
      { stopIndex: 0, field: "locationId" },
      { stopIndex: 0, field: "weight" },
      { stopIndex: 2, field: "scheduledWindowEnd" },
    ]);
  });

  it("puts errors on the move's own fields and on its list of stops on the move", () => {
    const stops = Object.assign([undefined, { type: required("Stop type is required") }], {
      root: required("A move needs a pickup and a delivery"),
    });
    const errors = { distance: required("Distance cannot be negative"), stops };

    expect(moveProblems(0, errors)).toEqual([
      {
        stopIndex: null,
        field: "distance",
        path: "moves.0.distance",
        message: "Distance cannot be negative",
      },
      {
        stopIndex: 1,
        field: "type",
        path: "moves.0.stops.1.type",
        message: "Stop type is required",
      },
      {
        stopIndex: null,
        field: "stops",
        path: "moves.0.stops",
        message: "A move needs a pickup and a delivery",
      },
    ]);
  });

  it("reads an error on the list of stops that has no item errors as the move's", () => {
    expect(moveProblems(1, { stops: required("Add at least two stops") })).toEqual([
      {
        stopIndex: null,
        field: "stops",
        path: "moves.1.stops",
        message: "Add at least two stops",
      },
    ]);
  });

  it("places an error nested inside a stop field on that stop, under the field", () => {
    const errors = { stops: [{ location: { name: required("Expected a string") } }] };

    expect(moveProblems(0, errors)).toEqual([
      {
        stopIndex: 0,
        field: "location",
        path: "moves.0.stops.0.location.name",
        message: "Expected a string",
      },
    ]);
  });

  it("finds nothing when the move has no errors", () => {
    expect(moveProblems(0, undefined)).toEqual([]);
    expect(moveProblems(0, {})).toEqual([]);
  });
});

describe("problem text", () => {
  const onLocation = moveProblems(0, {
    stops: [undefined, { locationId: required("Location is required") }],
  })[0]!;

  it("names the field before what is wrong with it", () => {
    expect(problemText(onLocation, translate)).toBe("Location: Location is required");
  });

  it("names the stop too when the list covers the whole move", () => {
    expect(problemTextInMove(onLocation, translate)).toBe(
      "Stop 2 · Location: Location is required",
    );
  });

  it("says a problem on a field it has no name for as the message alone", () => {
    const unnamed = moveProblems(0, {
      stops: [{ shipmentMoveId: required("Unknown move") }],
    })[0]!;

    expect(problemTextInMove(unnamed, translate)).toBe("Stop 1 · Unknown move");
  });
});
