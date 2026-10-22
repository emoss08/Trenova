import { rulesSync } from "@trenova/shared/rules/runtime";
import { laneKeys, parseLaneKeys } from "@trenova/shared/rules/lane";
import { describe, expect, it } from "vitest";
import { laneEndDisplay, laneKeyDisplay, laneSegmentsDisplay } from "../rate";
import type { RateAgreementRule, RateScopeType } from "../../types/rate";

/**
 * Builds a lane with only the fields the display reads, so a test states the
 * one thing it is about.
 */
function lane(overrides: Partial<RateAgreementRule> = {}): RateAgreementRule {
  return {
    label: "",
    status: "Active",
    originScopeType: "Any",
    originScopeValue: "",
    originCity: "",
    destinationScopeType: "Any",
    destinationScopeValue: "",
    destinationCity: "",
    direction: "Directional",
    formulaTemplateId: "ft_per_mile",
    freightClassSource: "Commodity",
    allowDeficitRating: true,
    daysOfWeek: 0,
    hazmatOnly: false,
    tempControlOnly: false,
    priority: 0,
    effectiveFrom: 0,
    breaks: [],
    ...overrides,
  } as RateAgreementRule;
}

function keysOf(rule: RateAgreementRule) {
  return laneKeys(rulesSync(), [rule])[0];
}

const labels: Record<string, string> = {
  "State:us_tx": "TX",
  "State:us_il": "IL",
  "CityState:us_tx": "TX",
  "CityState:us_il": "IL",
  "Zone:rzn_sw": "Southwest",
  "Location:loc_dal": "Dallas DC",
};
const resolve = (type: RateScopeType, value: string) => labels[`${type}:${value}`];

describe("laneKeyDisplay", () => {
  it("shows the record's name where the key stores its id", () => {
    const display = laneKeyDisplay(
      keysOf(
        lane({
          originScopeType: "CityState",
          originScopeValue: "us_tx",
          originCity: "Dallas",
          destinationScopeType: "CityState",
          destinationScopeValue: "us_il",
          destinationCity: "Chicago",
        }),
      ),
      resolve,
    );

    expect(display).toBe("CS:TX|DALLAS>CS:IL|CHICAGO");
    expect(display).not.toContain("us_");
  });

  it("resolves every id-backed scope: state, zone and location", () => {
    expect(
      laneKeyDisplay(
        keysOf(
          lane({
            originScopeType: "Zone",
            originScopeValue: "rzn_sw",
            destinationScopeType: "Location",
            destinationScopeValue: "loc_dal",
          }),
        ),
        resolve,
      ),
    ).toBe("ZN:Southwest>LOC:Dallas DC");

    expect(
      laneKeyDisplay(
        keysOf(lane({ originScopeType: "State", originScopeValue: "us_il" })),
        resolve,
      ),
    ).toBe("ST:IL>ANY");
  });

  // Literal scopes carry the value itself, so the display reads exactly as the
  // key does — including the postal folding.
  it("renders literal scopes the way the key stores them", () => {
    expect(
      laneKeyDisplay(
        keysOf(
          lane({
            originScopeType: "Zip3",
            originScopeValue: "60601-1234",
            destinationScopeType: "Country",
            destinationScopeValue: "usa",
          }),
        ),
        resolve,
      ),
    ).toBe("Z3:606>CT:USA");
  });

  // A name that has not arrived yet must read as pending, never as the raw id.
  it("shows a placeholder rather than the id while a name is unresolved", () => {
    const display = laneKeyDisplay(
      keysOf(lane({ originScopeType: "Zone", originScopeValue: "rzn_unknown" })),
      resolve,
    );

    expect(display).toBe("ZN:…>ANY");
    expect(display).not.toContain("rzn_unknown");
  });

  it("has no display key for a lane matched by radius, exactly like the key", () => {
    expect(laneKeyDisplay(keysOf(lane({ originScopeType: "Radius" })), resolve)).toBeNull();
  });

  // The engine stores no key for an end with no value, so the lane is matched by radius; the
  // old browser copy showed ">ANY" for it (docs/engineering/rules-engine.md, Lanes).
  it("has no display key for an end that names nothing, as the engine stores none", () => {
    expect(
      laneKeyDisplay(keysOf(lane({ originScopeType: "Location", originScopeValue: "" })), resolve),
    ).toBeNull();
  });
});

describe("laneSegmentsDisplay", () => {
  function display(key: string) {
    return laneSegmentsDisplay(parseLaneKeys(rulesSync(), [key])[0], resolve);
  }

  // Stored keys arrive from the server already assembled — import diffs and
  // simulation rows carry them as strings, with no rule object to read from.
  it("substitutes names for the ids a stored key carries", () => {
    const shown = display("CS:us_tx|DALLAS>CS:us_il|CHICAGO");
    expect(shown).toBe("CS:TX|DALLAS>CS:IL|CHICAGO");
    expect(shown).not.toContain("us_");
  });

  it("resolves zone, state and location ids", () => {
    expect(display("ZN:rzn_sw>LOC:loc_dal")).toBe("ZN:Southwest>LOC:Dallas DC");
    expect(display("ST:us_il>ANY")).toBe("ST:IL>ANY");
  });

  it("leaves literal segments exactly as stored", () => {
    expect(display("Z3:606>CT:USA")).toBe("Z3:606>CT:USA");
  });

  it("shows a placeholder rather than the id while a name is unresolved", () => {
    const shown = display("ZN:rzn_unknown>ANY");
    expect(shown).toBe("ZN:…>ANY");
    expect(shown).not.toContain("rzn_unknown");
  });

  it("passes the radius sentinel and empty keys through untouched", () => {
    expect(display("RADIUS")).toBe("RADIUS");
    expect(display("")).toBe("");
  });
});

describe("laneEndDisplay", () => {
  const names: Record<string, string> = {
    st_ga: "GA",
    st_tx: "TX",
    zn_se: "Southeast",
    loc_1: "Acme DC",
  };
  const byValue = (_type: RateScopeType, value: string) => names[value];
  const end = (rule: RateAgreementRule, side: "origin" | "destination") =>
    laneEndDisplay(rule, side, byValue, keysOf(rule));

  it("reads each kind of end the way a person names a place", () => {
    const rule = lane({
      originScopeType: "CityState",
      originScopeValue: "st_ga",
      originCity: "Atlanta",
      destinationScopeType: "State",
      destinationScopeValue: "st_tx",
    });

    expect(end(rule, "origin")).toBe("Atlanta, GA");
    expect(end(rule, "destination")).toBe("TX");
    expect(end(lane({ originScopeType: "Zone", originScopeValue: "zn_se" }), "origin")).toBe(
      "Southeast",
    );
    expect(end(lane({ originScopeType: "Location", originScopeValue: "loc_1" }), "origin")).toBe(
      "Acme DC",
    );
  });

  it("writes postal codes and countries as the engine keys them", () => {
    expect(end(lane({ originScopeType: "Zip5", originScopeValue: "30303-1234" }), "origin")).toBe(
      "ZIP 30303",
    );
    expect(end(lane({ originScopeType: "Zip3", originScopeValue: "30303" }), "origin")).toBe(
      "ZIP 303xx",
    );
    expect(end(lane({ originScopeType: "Country", originScopeValue: " mx " }), "origin")).toBe(
      "MX",
    );
    expect(end(lane(), "destination")).toBe("Anywhere");
  });

  // A postal prefix shorter than three characters has no key; the end shows what was typed.
  it("shows a postal end the engine cannot key as it was typed", () => {
    expect(end(lane({ originScopeType: "Zip3", originScopeValue: "30" }), "origin")).toBe(
      "ZIP 30xx",
    );
  });

  it("reads a radius around its city in miles", () => {
    expect(
      end(
        lane({ originScopeType: "Radius", originCity: "Memphis", originRadiusMeters: 80467.2 }),
        "origin",
      ),
    ).toBe("Within 50 mi of Memphis");
  });

  it("never shows an id whose name has not come back yet", () => {
    expect(end(lane({ originScopeType: "State", originScopeValue: "st_unknown" }), "origin")).toBe(
      "…",
    );
  });
});
