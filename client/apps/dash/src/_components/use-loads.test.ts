import type { PortalLoad, PortalStop } from "@trenova/shared/lib/graphql/driver-portal";
import { describe, expect, it } from "vitest";

import {
  destinationStop,
  directionsUrl,
  formatMiles,
  isLikelyUSAddress,
  originStop,
  stopPlace,
} from "./use-loads";

function stop(overrides: Partial<PortalStop>): PortalStop {
  return { locationName: "", addressLine: "", ...overrides } as PortalStop;
}

function load(stops: PortalStop[]): PortalLoad {
  return { stops } as PortalLoad;
}

describe("origin and destination stops", () => {
  const pickup = stop({ locationName: "Origin DC" });
  const delivery = stop({ locationName: "Destination DC" });

  it("uses the first stop as origin", () => {
    expect(originStop(load([pickup, delivery]))).toBe(pickup);
  });

  it("uses the last stop as destination only when there is more than one", () => {
    expect(destinationStop(load([pickup, delivery]))).toBe(delivery);
    expect(destinationStop(load([pickup]))).toBeUndefined();
    expect(destinationStop(load([]))).toBeUndefined();
  });
});

describe("stopPlace", () => {
  it("prefers the location name, then the address, then a dash", () => {
    expect(stopPlace(stop({ locationName: "Yard 4", addressLine: "1 Main St" }))).toBe("Yard 4");
    expect(stopPlace(stop({ addressLine: "1 Main St" }))).toBe("1 Main St");
    expect(stopPlace(stop({}))).toBe("—");
    expect(stopPlace(undefined)).toBe("—");
  });
});

describe("isLikelyUSAddress", () => {
  it("matches state and zip endings", () => {
    expect(isLikelyUSAddress("123 Main St, Atlanta, GA 30303")).toBe(true);
    expect(isLikelyUSAddress("123 Main St, Atlanta, GA 30303-1234")).toBe(true);
    expect(isLikelyUSAddress("123 Main St, Atlanta, GA 30303, USA")).toBe(true);
  });

  it("rejects non-US shapes", () => {
    expect(isLikelyUSAddress("10 Downing Street, London SW1A 2AA")).toBe(false);
    expect(isLikelyUSAddress("Somewhere")).toBe(false);
  });
});

describe("directionsUrl", () => {
  it("routes US addresses to TruckMap", () => {
    const url = directionsUrl(stop({ addressLine: "123 Main St, Atlanta, GA 30303" }));
    expect(url).toContain("truckmap.com/search/");
    expect(url).toContain(encodeURIComponent("123 Main St, Atlanta, GA 30303"));
  });

  it("routes everything else to Google Maps", () => {
    const url = directionsUrl(stop({ locationName: "Toronto Yard" }));
    expect(url).toContain("google.com/maps/dir");
    expect(url).toContain(encodeURIComponent("Toronto Yard"));
  });
});

describe("formatMiles", () => {
  it("formats positive mileage with grouping", () => {
    expect(formatMiles(1234.6)).toBe("1,235 mi");
    expect(formatMiles(1)).toBe("1 mi");
  });

  it("returns null for missing or non-positive values", () => {
    expect(formatMiles(0)).toBeNull();
    expect(formatMiles(-5)).toBeNull();
    expect(formatMiles(null)).toBeNull();
    expect(formatMiles(undefined)).toBeNull();
  });
});
