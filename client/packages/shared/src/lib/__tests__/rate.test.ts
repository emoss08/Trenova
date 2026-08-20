import { describe, expect, it } from "vitest";
import {
  findCoverageIssues,
  laneKeyDisplay,
  laneKeyPreview,
  laneKeyScopeIds,
  laneKeyStringDisplay,
  laneSpecificity,
} from "../rate";
import type { RateAgreementRule, RateScopeType } from "../../types/rate";

/**
 * Builds a lane with only the fields the scoring reads, so a test states the
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

describe("laneSpecificity", () => {
  it("scores a lane that matches anywhere at zero", () => {
    expect(laneSpecificity(lane())).toBe(0);
  });

  // The editor predicts the winner as somebody types. If it ranked scopes
  // differently from the engine, it would show one lane winning and the invoice
  // would show another.
  it("ranks narrower geography above wider geography", () => {
    const location = laneSpecificity(
      lane({ originScopeType: "Location", destinationScopeType: "Location" }),
    );
    const city = laneSpecificity(
      lane({ originScopeType: "CityState", destinationScopeType: "CityState" }),
    );
    const zone = laneSpecificity(lane({ originScopeType: "Zone", destinationScopeType: "Zone" }));
    const state = laneSpecificity(
      lane({ originScopeType: "State", destinationScopeType: "State" }),
    );

    expect(location).toBeGreaterThan(city);
    expect(city).toBeGreaterThan(zone);
    expect(zone).toBeGreaterThan(state);
    expect(state).toBeGreaterThan(0);
  });

  // The two ends are weighted equally on purpose. An origin-specific rule and a
  // destination-specific rule of the same granularity are equally narrow, and
  // there is no principled reason to prefer one — the engine says so, and the
  // editor must not disagree.
  it("weights the two ends of a lane equally", () => {
    const narrowOrigin = laneSpecificity(
      lane({ originScopeType: "Location", destinationScopeType: "Any" }),
    );
    const narrowDestination = laneSpecificity(
      lane({ originScopeType: "Any", destinationScopeType: "Location" }),
    );

    expect(narrowOrigin).toBe(narrowDestination);
  });

  // Geography is scaled clear of the conditions so that a lane written to a
  // tighter place always beats a wider one, however many conditions the wider
  // one piles on. Without that, a state lane carrying every condition would
  // outrank a city lane carrying none.
  it("never lets conditions outweigh geography", () => {
    // Every condition the scoring knows about, on the widest possible lane.
    const everyCondition = laneSpecificity(
      lane({
        commodityIds: ["c1"],
        freightClasses: ["70"],
        tractorTypeIds: ["t1"],
        trailerTypeIds: ["t2"],
        equipmentClasses: ["Van"],
        serviceTypeIds: ["s1"],
        shipmentTypeIds: ["st1"],
        serviceModels: ["Truckload"],
        minWeight: 1000,
        maxWeight: 40000,
        minDistance: 100,
        maxDistance: 900,
        hazmatOnly: true,
        tempControlOnly: true,
      }),
    );

    // One step of geographic granularity, with no conditions at all.
    const oneStepNarrower = laneSpecificity(lane({ originScopeType: "Country" }));

    expect(oneStepNarrower).toBeGreaterThan(everyCondition);
  });

  it("scores commodity and freight class as separate narrowings", () => {
    const bare = laneSpecificity(lane({ originScopeType: "State" }));
    const withCommodity = laneSpecificity(lane({ originScopeType: "State", commodityIds: ["c1"] }));
    const withClassToo = laneSpecificity(
      lane({ originScopeType: "State", commodityIds: ["c1"], freightClasses: ["70"] }),
    );

    expect(withCommodity).toBeGreaterThan(bare);
    expect(withClassToo).toBeGreaterThan(withCommodity);
  });

  // Naming a tractor type, a trailer type and an equipment class is one
  // statement about equipment, not three.
  it("counts every way of naming equipment as one narrowing", () => {
    const one = laneSpecificity(lane({ tractorTypeIds: ["t1"] }));
    const all = laneSpecificity(
      lane({ tractorTypeIds: ["t1"], trailerTypeIds: ["t2"], equipmentClasses: ["Van"] }),
    );

    expect(all).toBe(one);
  });

  it("treats an empty condition list as no condition at all", () => {
    expect(laneSpecificity(lane({ serviceTypeIds: [] }))).toBe(
      laneSpecificity(lane({ serviceTypeIds: null })),
    );
  });
});

describe("laneKeyPreview", () => {
  it("renders a lane that matches anywhere", () => {
    expect(laneKeyPreview(lane())).toBe("ANY>ANY");
  });

  it("prefixes each scope the way the engine stores it", () => {
    const key = laneKeyPreview(
      lane({
        originScopeType: "Zone",
        originScopeValue: "rzn_sw",
        destinationScopeType: "State",
        destinationScopeValue: "us_il",
      }),
    );

    expect(key).toBe("ZN:rzn_sw>ST:us_il");
  });

  // A city typed three different ways has to reach the same rate.
  it("folds a city before it becomes part of a key", () => {
    const lower = laneKeyPreview(
      lane({
        originScopeType: "CityState",
        originScopeValue: "us_tx",
        originCity: " dallas ",
      }),
    );
    const upper = laneKeyPreview(
      lane({
        originScopeType: "CityState",
        originScopeValue: "us_tx",
        originCity: "DALLAS",
      }),
    );

    expect(lower).toBe(upper);
    expect(lower).toBe("CS:us_tx|DALLAS>ANY");
  });

  it("takes only the first three digits of a postal prefix", () => {
    expect(laneKeyPreview(lane({ originScopeType: "Zip3", originScopeValue: "60601-1234" }))).toBe(
      "Z3:606>ANY",
    );
  });

  // A radius lane is found through a geospatial index, so it has no key at all
  // and the editor must say so rather than render a misleading one.
  it("has no key for a lane matched by radius", () => {
    expect(laneKeyPreview(lane({ originScopeType: "Radius" }))).toBeNull();
    expect(laneKeyPreview(lane({ destinationScopeType: "Radius" }))).toBeNull();
  });
});

describe("laneKeyDisplay", () => {
  const labels: Record<string, string> = {
    "State:us_tx": "TX",
    "State:us_il": "IL",
    "CityState:us_tx": "TX",
    "CityState:us_il": "IL",
    "Zone:rzn_sw": "Southwest",
    "Location:loc_dal": "Dallas DC",
  };
  const resolve = (type: RateScopeType, value: string) => labels[`${type}:${value}`];

  it("shows the record's name where the key stores its id", () => {
    const display = laneKeyDisplay(
      lane({
        originScopeType: "CityState",
        originScopeValue: "us_tx",
        originCity: "Dallas",
        destinationScopeType: "CityState",
        destinationScopeValue: "us_il",
        destinationCity: "Chicago",
      }),
      resolve,
    );

    expect(display).toBe("CS:TX|DALLAS>CS:IL|CHICAGO");
    expect(display).not.toContain("us_");
  });

  it("resolves every id-backed scope: state, zone and location", () => {
    expect(
      laneKeyDisplay(
        lane({
          originScopeType: "Zone",
          originScopeValue: "rzn_sw",
          destinationScopeType: "Location",
          destinationScopeValue: "loc_dal",
        }),
        resolve,
      ),
    ).toBe("ZN:Southwest>LOC:Dallas DC");

    expect(
      laneKeyDisplay(lane({ originScopeType: "State", originScopeValue: "us_il" }), resolve),
    ).toBe("ST:IL>ANY");
  });

  // Literal scopes carry the value itself, so the display reads exactly as the
  // key does — including the postal folding.
  it("renders literal scopes the way the key stores them", () => {
    expect(
      laneKeyDisplay(
        lane({
          originScopeType: "Zip3",
          originScopeValue: "60601-1234",
          destinationScopeType: "Country",
          destinationScopeValue: "usa",
        }),
        resolve,
      ),
    ).toBe("Z3:606>CT:USA");
  });

  // A name that has not arrived yet must read as pending, never as the raw id
  // — the id is exactly what this display exists to avoid.
  it("shows a placeholder rather than the id while a name is unresolved", () => {
    const display = laneKeyDisplay(
      lane({ originScopeType: "Zone", originScopeValue: "rzn_unknown" }),
      resolve,
    );

    expect(display).toBe("ZN:…>ANY");
    expect(display).not.toContain("rzn_unknown");
  });

  it("has no display key for a lane matched by radius, exactly like the key", () => {
    expect(laneKeyDisplay(lane({ originScopeType: "Radius" }), resolve)).toBeNull();
  });
});

describe("laneKeyStringDisplay", () => {
  const labels: Record<string, string> = {
    "State:us_tx": "TX",
    "State:us_il": "IL",
    "CityState:us_tx": "TX",
    "CityState:us_il": "IL",
    "Zone:rzn_sw": "Southwest",
    "Location:loc_dal": "Dallas DC",
  };
  const resolve = (type: RateScopeType, value: string) => labels[`${type}:${value}`];

  // Stored keys arrive from the server already assembled — import diffs and
  // simulation rows carry them as strings, with no rule object to read from.
  it("substitutes names for the ids a stored key carries", () => {
    const display = laneKeyStringDisplay("CS:us_tx|DALLAS>CS:us_il|CHICAGO", resolve);

    expect(display).toBe("CS:TX|DALLAS>CS:IL|CHICAGO");
    expect(display).not.toContain("us_");
  });

  it("resolves zone, state and location ids", () => {
    expect(laneKeyStringDisplay("ZN:rzn_sw>LOC:loc_dal", resolve)).toBe(
      "ZN:Southwest>LOC:Dallas DC",
    );
    expect(laneKeyStringDisplay("ST:us_il>ANY", resolve)).toBe("ST:IL>ANY");
  });

  it("leaves literal segments exactly as stored", () => {
    expect(laneKeyStringDisplay("Z3:606>CT:USA", resolve)).toBe("Z3:606>CT:USA");
  });

  it("shows a placeholder rather than the id while a name is unresolved", () => {
    const display = laneKeyStringDisplay("ZN:rzn_unknown>ANY", resolve);

    expect(display).toBe("ZN:…>ANY");
    expect(display).not.toContain("rzn_unknown");
  });

  it("passes the radius sentinel and empty keys through untouched", () => {
    expect(laneKeyStringDisplay("RADIUS", resolve)).toBe("RADIUS");
    expect(laneKeyStringDisplay("", resolve)).toBe("");
  });
});

describe("laneKeyScopeIds", () => {
  it("lists every id a stored key references, typed by scope", () => {
    expect(laneKeyScopeIds("CS:us_tx|DALLAS>ZN:rzn_sw")).toEqual([
      { type: "CityState", id: "us_tx" },
      { type: "Zone", id: "rzn_sw" },
    ]);
  });

  it("finds nothing in keys made only of literals", () => {
    expect(laneKeyScopeIds("Z3:606>CT:USA")).toEqual([]);
    expect(laneKeyScopeIds("ANY>ANY")).toEqual([]);
    expect(laneKeyScopeIds("RADIUS")).toEqual([]);
  });
});

describe("findCoverageIssues", () => {
  it("finds nothing wrong with lanes written at different widths", () => {
    const rules = [
      lane({ originScopeType: "CityState", originScopeValue: "us_tx", originCity: "Dallas" }),
      lane({ originScopeType: "State", originScopeValue: "us_tx" }),
      lane(),
    ];

    expect(findCoverageIssues(rules)).toEqual([]);
  });

  // Two lanes written exactly as narrowly leave the winner to a tie-break,
  // which is to say one of them silently never applies.
  it("reports two lanes that are equally narrow as a coin toss", () => {
    const rules = [
      lane({ label: "first", originScopeType: "State", originScopeValue: "us_tx" }),
      lane({ label: "second", originScopeType: "State", originScopeValue: "us_tx" }),
    ];

    const issues = findCoverageIssues(rules);

    expect(issues).toHaveLength(1);
    expect(issues[0].index).toBe(1);
    expect(issues[0].kind).toBe("duplicate");
  });

  it("reports a lane sitting behind an identical one with a higher priority", () => {
    const rules = [
      lane({ originScopeType: "State", originScopeValue: "us_tx", priority: 10 }),
      lane({ originScopeType: "State", originScopeValue: "us_tx", priority: 0 }),
    ];

    const issues = findCoverageIssues(rules);

    expect(issues).toHaveLength(1);
    expect(issues[0].kind).toBe("shadowed");
  });

  // Two lanes on the same geography that narrow on different conditions are a
  // legitimate pair, not a conflict: the engine ranks them apart.
  it("does not fault two lanes that narrow on different conditions", () => {
    const rules = [
      lane({ originScopeType: "State", originScopeValue: "us_tx", commodityIds: ["c1"] }),
      lane({ originScopeType: "State", originScopeValue: "us_tx" }),
    ];

    expect(findCoverageIssues(rules)).toEqual([]);
  });

  it("ignores an inactive lane, which cannot conflict with anything", () => {
    const rules = [
      lane({ originScopeType: "State", originScopeValue: "us_tx" }),
      lane({ originScopeType: "State", originScopeValue: "us_tx", status: "Inactive" }),
    ];

    expect(findCoverageIssues(rules)).toEqual([]);
  });

  // Radius overlap cannot be judged from the fields alone, so claiming a
  // conflict would be a guess presented as a fact.
  it("does not judge overlap between radius lanes", () => {
    const rules = [
      lane({ originScopeType: "Radius", originRadiusMeters: 80000 }),
      lane({ originScopeType: "Radius", originRadiusMeters: 80000 }),
    ];

    expect(findCoverageIssues(rules)).toEqual([]);
  });
});
