import type { DeskLocation, LocationDwell, LocationPolicy } from "@/lib/graphql/location-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { outlineReach } from "../../../record-card/map-tiles";
import {
  dwellMoneyLine,
  dwellStats,
  dwellWindow,
  facilityTerms,
  geofenceLine,
  geofenceOutline,
  LOCATION_ZOOM,
  locationFacts,
  locationStanding,
  locationZoom,
  policyFreeTime,
  policyScope,
  streetLine,
} from "../location-card";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const NOW = 1_791_720_000;
const DAY = 86_400;

function location(overrides: Partial<DeskLocation> = {}): DeskLocation {
  return {
    id: "loc_1",
    status: "Active",
    code: "DAL-YD",
    name: "Dallas yard",
    description: "",
    addressLine1: "1801 Commerce St",
    addressLine2: "",
    city: "Dallas",
    postalCode: "75201",
    timezone: "America/Chicago",
    isGeocoded: true,
    latitude: 32.7801,
    longitude: -96.8005,
    geofenceType: "circle",
    geofenceRadiusMeters: 250,
    geofenceVertices: [],
    createdAt: NOW - 400 * DAY,
    state: { abbreviation: "TX" },
    locationCategory: {
      id: "lc_1",
      name: "Company yard",
      type: "Terminal",
      facilityType: null,
      hasSecureParking: true,
      requiresAppointment: false,
      allowsOvernight: true,
      hasRestroom: true,
    },
    ...overrides,
  } as DeskLocation;
}

function dwell(overrides: Partial<LocationDwell> = {}): LocationDwell {
  return {
    locationId: "loc_1",
    stopCount: 40,
    breachCount: 6,
    avgDwellMinutes: 96,
    medianDwellMinutes: 80,
    p90DwellMinutes: 150,
    billedAmount: "1250",
    waivedAmount: "0",
    disputeCount: 0,
    ...overrides,
  } as LocationDwell;
}

function policy(overrides: Partial<LocationPolicy> = {}): LocationPolicy {
  return {
    id: "dp_1",
    name: "Dallas yard terms",
    code: "DAL",
    status: "Active",
    priority: 10,
    customerId: null,
    stopTypes: [],
    appointmentStopsOnly: false,
    billingFreeMinutes: 120,
    pickupFreeMinutes: null,
    deliveryFreeMinutes: null,
    effectiveStartDate: null,
    effectiveEndDate: null,
    ...overrides,
  } as LocationPolicy;
}

describe("dwellWindow", () => {
  it("reads ninety days to the top of the hour, so a card opened in the same hour shares it", () => {
    const window = dwellWindow(NOW + 1_234);
    expect(window.to).toBe(NOW);
    expect(window.to - window.from).toBe(90 * DAY);
    expect(dwellWindow(NOW + 3_599)).toEqual(window);
  });
});

describe("address lines", () => {
  it("joins the street lines", () => {
    expect(streetLine(location({ addressLine2: " Gate 4 " }))).toBe("1801 Commerce St, Gate 4");
  });

  it("leaves out what is not on file", () => {
    expect(streetLine(location({ addressLine1: " ", addressLine2: "" }))).toBe("");
  });
});

describe("locationStanding", () => {
  it("reads an active location as taking trucks and an inactive one as muted", () => {
    expect(locationStanding("Active", t)).toEqual({ label: "Active", tone: "success" });
    expect(locationStanding("Inactive", t)).toEqual({ label: "Inactive", tone: "muted" });
  });
});

describe("facilityTerms", () => {
  it("names what the facility offers, saying whether it takes an appointment and an overnight", () => {
    expect(facilityTerms(location().locationCategory, t)).toEqual([
      "No appointment needed",
      "Secure parking",
      "Overnight allowed",
      "Restroom",
    ]);
    expect(
      facilityTerms(
        {
          ...location().locationCategory!,
          requiresAppointment: true,
          hasSecureParking: false,
          allowsOvernight: false,
          hasRestroom: false,
        },
        t,
      ),
    ).toEqual(["Appointment required", "No overnight"]);
    expect(facilityTerms(null, t)).toEqual([]);
  });
});

describe("geofence", () => {
  it("draws a circle or an automatic fence by its radius", () => {
    expect(geofenceOutline(location())).toEqual({ kind: "circle", radiusMeters: 250 });
    expect(geofenceOutline(location({ geofenceType: "auto", geofenceRadiusMeters: 120 }))).toEqual({
      kind: "circle",
      radiusMeters: 120,
    });
    expect(geofenceLine(location({ geofenceRadiusMeters: 1250 }), t)).toBe("Circle fence, 1,250 m");
    expect(geofenceLine(location({ geofenceType: "auto", geofenceRadiusMeters: null }), t)).toBe(
      "Automatic fence",
    );
  });

  it("draws a drawn or rectangle fence by its corners, and nothing with fewer than three", () => {
    const vertices = [
      { latitude: 32.781, longitude: -96.802 },
      { latitude: 32.781, longitude: -96.799 },
      { latitude: 32.779, longitude: -96.799 },
    ];
    expect(geofenceOutline(location({ geofenceType: "draw", geofenceVertices: vertices }))).toEqual(
      { kind: "polygon", vertices },
    );
    expect(geofenceLine(location({ geofenceType: "draw", geofenceVertices: vertices }), t)).toBe(
      "Drawn fence, 3 corners",
    );
    expect(
      geofenceOutline(location({ geofenceType: "rectangle", geofenceVertices: vertices.slice(2) })),
    ).toBeNull();
    expect(geofenceOutline(location({ geofenceRadiusMeters: 0 }))).toBeNull();
  });

  it("reads the map as close as the fence allows, or the block when there is no fence", () => {
    const center = { latitude: 32.7801, longitude: -96.8005 };
    const outline = { kind: "circle", radiusMeters: 250 } as const;
    const zoom = locationZoom(center, outline, 112);

    expect(outlineReach(center, zoom, outline) * 2).toBeLessThanOrEqual(112);
    expect(zoom).toBeGreaterThan(11);
    expect(locationZoom(center, null, 112)).toBe(LOCATION_ZOOM);
  });
});

describe("dwell", () => {
  it("reads the waits, marking a long tail and a dock that keeps trucks past free time", () => {
    expect(dwellStats(dwell(), t).map((stat) => [stat.label, stat.value, stat.state])).toEqual([
      ["Average", "1h 36m", undefined],
      ["Median", "1h 20m", undefined],
      ["90th percentile", "2h 30m", "ok"],
      ["Past free time", "6 of 40", "ok"],
    ]);
    const long = dwellStats(dwell({ p90DwellMinutes: 181, breachCount: 10 }), t);
    expect(long.map((stat) => stat.state)).toEqual([undefined, undefined, "warn", "warn"]);
  });

  it("reads what the detention came to, naming waivers and disputes only when there are any", () => {
    expect(dwellMoneyLine(dwell(), t)).toBe("$1,250.00 detention billed on 40 stops");
    expect(dwellMoneyLine(dwell({ waivedAmount: "75", disputeCount: 1, stopCount: 1 }), t)).toBe(
      "$1,250.00 detention billed on 1 stop · $75.00 waived · 1 dispute",
    );
  });
});

describe("policies", () => {
  it("reads the free time at pickup and delivery, falling back to the billing free time", () => {
    expect(policyFreeTime(policy(), t)).toBe("2h free");
    expect(policyFreeTime(policy({ pickupFreeMinutes: 60 }), t)).toBe(
      "1h at pickup · 2h at delivery",
    );
  });

  it("says who and what the policy covers", () => {
    expect(policyScope(policy(), t)).toBe("Every customer");
    expect(
      policyScope(
        policy({
          customerId: "cus_1",
          stopTypes: ["Pickup", "SplitPickup"],
          appointmentStopsOnly: true,
          effectiveEndDate: NOW,
        }),
        t,
      ),
    ).toMatch(/^One customer · Pickup, Split pickup · Appointment stops only · Through /u);
  });
});

describe("locationFacts", () => {
  it("lists the code, category, timezone and coordinates, saying what is not on file", () => {
    const facts = Object.fromEntries(
      locationFacts(
        location({ timezone: "", latitude: null, longitude: null, locationCategory: null }),
        t,
      ).map((fact) => [fact.key, fact.value]),
    );
    expect(facts).toMatchObject({
      code: "DAL-YD",
      category: "None",
      timezone: "None on file",
      coordinates: "Not geocoded",
      description: "None",
    });
    expect(locationFacts(location(), t).find((fact) => fact.key === "coordinates")?.value).toBe(
      "32.78010, -96.80050",
    );
  });
});
