import type { DeskCaseRoute } from "@/lib/graphql/desk-case";
import { describe, expect, it } from "vitest";
import { routeHasCoordinates } from "../case-route";

function route(overrides: Partial<DeskCaseRoute> = {}): DeskCaseRoute {
  return {
    points: [
      { label: "Dallas", x: 0, y: 1, latitude: 32.78, longitude: -96.8, state: "Done" },
      { label: "Chicago", x: 1, y: 0, latitude: 41.88, longitude: -87.63, state: "Ahead" },
    ],
    truck: { x: 0.7, y: 0.3, latitude: 39.78, longitude: -89.65 },
    callout: "",
    calloutTone: "Neutral",
    ...overrides,
  } as DeskCaseRoute;
}

/** The map is drawn only from places the server sent; without them the sketch is. */
describe("routeHasCoordinates", () => {
  it("is true when every stop and the truck have a place", () => {
    expect(routeHasCoordinates(route())).toBe(true);
    expect(routeHasCoordinates(route({ truck: null }))).toBe(true);
  });

  it("is false when a stop or the truck came without one", () => {
    const [first, second] = route().points;
    if (!first || !second) {
      throw new Error("the fixture route has two stops");
    }

    expect(
      routeHasCoordinates(route({ points: [first, { ...second, latitude: 0, longitude: 0 }] })),
    ).toBe(false);
    expect(routeHasCoordinates(route({ truck: { x: 0, y: 0, latitude: 0, longitude: 0 } }))).toBe(
      false,
    );
    expect(routeHasCoordinates(route({ points: [] }))).toBe(false);
  });
});
