import { describe, expect, it } from "vitest";
import {
  EMPTY_SOURCING_FILTERS,
  buildSourcingSearchInput,
  detectSourcingIntent,
  lookupInputForIntent,
  type AuthorityAgePreset,
} from "../carrier-sourcing";

describe("detectSourcingIntent", () => {
  it("treats blank input as empty", () => {
    expect(detectSourcingIntent("   ")).toEqual({ kind: "empty" });
  });

  it.each([
    ["265752", "265752"],
    [" 1234567 ", "1234567"],
    ["USDOT 265752", "265752"],
    ["usdot265752", "265752"],
    ["DOT# 80806", "80806"],
    ["US DOT: 12345678", "12345678"],
  ])("reads %j as a USDOT lookup", (input, dotNumber) => {
    expect(detectSourcingIntent(input)).toEqual({ kind: "dot", dotNumber });
  });

  it.each([
    ["MC123456", "123456"],
    ["mc 179059", "179059"],
    ["MC-42", "42"],
  ])("reads %j as an MC lookup", (input, docketNumber) => {
    expect(detectSourcingIntent(input)).toEqual({ kind: "mc", docketNumber });
  });

  it("builds the lookup input for each number kind", () => {
    expect(lookupInputForIntent({ kind: "dot", dotNumber: "265752" }, "Lite")).toEqual({
      dotNumber: "265752",
      depth: "Lite",
    });
    expect(lookupInputForIntent({ kind: "mc", docketNumber: "179059" }, null)).toEqual({
      docketNumber: "179059",
      depth: null,
    });
  });

  it.each(["123456789", "12-3456789"])("reads %j as an EIN search", (input) => {
    expect(detectSourcingIntent(input)).toEqual({ kind: "ein", text: input });
  });

  it("reads a 17-character VIN and upper-cases it", () => {
    expect(detectSourcingIntent("1fuja6cv74lm12345")).toEqual({
      kind: "vin",
      text: "1FUJA6CV74LM12345",
    });
  });

  it.each([
    "Blue Ridge Freight",
    "ABCDEFGHJKLMNPRST",
    "1234567890",
    "MC Trucking",
    "DOT Logistics",
    "12-34",
  ])("reads %j as a name search", (input) => {
    expect(detectSourcingIntent(input)).toEqual({ kind: "name", text: input });
  });
});

describe("buildSourcingSearchInput", () => {
  it("does not search without text or a location", () => {
    expect(buildSourcingSearchInput("  ", EMPTY_SOURCING_FILTERS, "BestMatch")).toBeNull();
    expect(
      buildSourcingSearchInput("", { ...EMPTY_SOURCING_FILTERS, powerUnits: "11-50" }, "BestMatch"),
    ).toBeNull();
  });

  it("maps filters onto the search input", () => {
    expect(
      buildSourcingSearchInput(
        "",
        {
          state: "TX",
          originState: "IL",
          destinationState: "GA",
          powerUnits: "51-250",
          authorityAge: "3-5",
          screens: ["hazmat", "hideExisting"],
        },
        "FleetSizeDesc",
      ),
    ).toEqual({
      text: null,
      state: "TX",
      originState: "IL",
      destinationState: "GA",
      minPowerUnits: 51,
      maxPowerUnits: 250,
      minAuthorityAgeDays: 1095,
      maxAuthorityAgeDays: 1824,
      hazmatOnly: true,
      excludeBlocking: false,
      excludeExistingCarriers: true,
      sort: "FleetSizeDesc",
      limit: 25,
    });
  });

  it("leaves the maximum open for the largest fleets", () => {
    const input = buildSourcingSearchInput(
      "Werner",
      { ...EMPTY_SOURCING_FILTERS, powerUnits: "251+" },
      "BestMatch",
    );
    expect(input?.text).toBe("Werner");
    expect(input?.minPowerUnits).toBe(251);
    expect(input?.maxPowerUnits).toBeNull();
  });

  it.each<[AuthorityAgePreset, number | null, number | null]>([
    ["under1", null, 364],
    ["1-3", 365, 1094],
    ["3-5", 1095, 1824],
    ["5+", 1825, null],
  ])("bounds the %s authority age preset on both ends", (authorityAge, min, max) => {
    const input = buildSourcingSearchInput(
      "Werner",
      { ...EMPTY_SOURCING_FILTERS, authorityAge },
      "AuthorityAgeDesc",
    );
    expect(input?.minAuthorityAgeDays).toBe(min);
    expect(input?.maxAuthorityAgeDays).toBe(max);
    expect(input?.sort).toBe("AuthorityAgeDesc");
  });
});
