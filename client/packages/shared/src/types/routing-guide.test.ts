import { describe, expect, it } from "vitest";
import {
  deriveLaneMatchMode,
  emptyRoutingGuideEntry,
  emptyRoutingGuidePayload,
  formatRoutingGuideLane,
  routingGuidePayloadSchema,
} from "./routing-guide";

function makeGuidePayload(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    ...emptyRoutingGuidePayload,
    name: "Dallas → Atlanta",
    originCity: "Dallas",
    originState: "TX",
    destinationCity: "Atlanta",
    destinationState: "GA",
    entries: [{ ...emptyRoutingGuideEntry(1), carrierId: "car_01", rate: 1500 }],
    ...overrides,
  };
}

describe("routingGuidePayloadSchema", () => {
  it("accepts a city+state lane with a ranked entry", () => {
    const parsed = routingGuidePayloadSchema.parse(makeGuidePayload());
    expect(parsed.entries[0].rate).toBe(1500);
    expect(parsed.entries[0].rank).toBe(1);
  });

  it("coerces decimal-string rates loaded from GraphQL rows", () => {
    const parsed = routingGuidePayloadSchema.parse(
      makeGuidePayload({
        entries: [{ ...emptyRoutingGuideEntry(1), carrierId: "car_01", rate: "1500.2500" }],
      }),
    );
    expect(parsed.entries[0].rate).toBe(1500.25);
  });

  it("rejects a lane whose sides sit at different tiers, mirroring the server", () => {
    const result = routingGuidePayloadSchema.safeParse(
      makeGuidePayload({
        destinationCity: "",
        // Destination drops to state-only while origin stays city+state.
      }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      const issue = result.error.issues.find((item) =>
        item.message.includes("same level of detail"),
      );
      expect(issue?.path).toEqual(["originLocationId"]);
    }
  });

  it("rejects an empty lane side", () => {
    const result = routingGuidePayloadSchema.safeParse(
      makeGuidePayload({ originCity: "", originState: "" }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(
        result.error.issues.some(
          (item) => item.message === "Origin requires a location, a city and state, or a state",
        ),
      ).toBe(true);
    }
  });

  it("rejects a location mixed with city or state criteria", () => {
    const result = routingGuidePayloadSchema.safeParse(
      makeGuidePayload({
        originLocationId: "loc_01",
        destinationLocationId: "loc_02",
      }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(
        result.error.issues.some((item) =>
          item.message.includes("cannot be combined with city or state"),
        ),
      ).toBe(true);
    }
  });

  it("requires at least one entry", () => {
    expect(() => routingGuidePayloadSchema.parse(makeGuidePayload({ entries: [] }))).toThrowError(
      /At least one carrier entry is required/,
    );
  });

  it("flags a duplicate rank on the offending entry, mirroring the server's entries[i].rank path", () => {
    const result = routingGuidePayloadSchema.safeParse(
      makeGuidePayload({
        entries: [
          { ...emptyRoutingGuideEntry(1), carrierId: "car_01", rate: 1000 },
          { ...emptyRoutingGuideEntry(1), carrierId: "car_02", rate: 1200 },
        ],
      }),
    );
    expect(result.success).toBe(false);
    if (!result.success) {
      const issue = result.error.issues.find(
        (item) => item.message === "Rank must be unique within the guide",
      );
      expect(issue?.path).toEqual(["entries", 1, "rank"]);
    }
  });
});

describe("deriveLaneMatchMode", () => {
  it("prefers locations, then city+state, then state-only", () => {
    expect(deriveLaneMatchMode({ originLocationId: "loc_01" })).toBe("locations");
    expect(deriveLaneMatchMode({ originCity: "Dallas" })).toBe("cityState");
    expect(deriveLaneMatchMode({})).toBe("stateOnly");
  });
});

describe("formatRoutingGuideLane", () => {
  it("formats each tier the way the row shows it", () => {
    expect(
      formatRoutingGuideLane({
        originCity: "Dallas",
        originState: "TX",
        destinationCity: "Atlanta",
        destinationState: "GA",
      }),
    ).toBe("Dallas, TX → Atlanta, GA");
    expect(formatRoutingGuideLane({ originState: "TX", destinationState: "GA" })).toBe("TX → GA");
    expect(
      formatRoutingGuideLane({ originLocationId: "loc_01", destinationLocationId: "loc_02" }),
    ).toBe("Exact location → Exact location");
  });
});
