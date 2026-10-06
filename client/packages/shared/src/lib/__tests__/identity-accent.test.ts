import { describe, expect, it } from "vitest";
import { BADGE_ACCENTS } from "../../types/badge";
import { identityAccent, identityAccentClass } from "../identity-accent";

describe("identityAccent", () => {
  it("gives the same record the same accent every time", () => {
    expect(identityAccent("wrk_01J8Z")).toBe(identityAccent("wrk_01J8Z"));
  });

  it("only ever picks a categorical accent", () => {
    for (const id of ["a", "b", "car_1", "wrk_2", "", "Knight-Swift"]) {
      expect(BADGE_ACCENTS).toContain(identityAccent(id));
    }
  });

  it("spreads different records across the set", () => {
    const seen = new Set(Array.from({ length: 64 }, (_, i) => identityAccent(`worker-${i}`)));
    expect(seen.size).toBeGreaterThan(4);
  });

  it("fills with the accent's subtle pair", () => {
    const accent = identityAccent("wrk_01J8Z");
    expect(identityAccentClass("wrk_01J8Z")).toBe(`bg-${accent}-subtle text-${accent}-on-subtle`);
  });
});
