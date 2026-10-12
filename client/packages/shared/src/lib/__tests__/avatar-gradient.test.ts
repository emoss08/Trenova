import { describe, expect, it } from "vitest";
import {
  AVATAR_PRESETS,
  avatarBackground,
  avatarMesh,
  avatarSeedHues,
  fnv1a,
  isAvatarStyle,
} from "../avatar-gradient";

describe("fnv1a", () => {
  it("matches the published FNV-1a 32-bit vectors", () => {
    expect(fnv1a("")).toBe(0x811c9dc5);
    expect(fnv1a("a")).toBe(0xe40c292c);
    expect(fnv1a("foobar")).toBe(0xbf9cf968);
  });
});

describe("avatarSeedHues", () => {
  it("is stable for one username and different for another", () => {
    expect(avatarSeedHues("maya.chen")).toEqual(avatarSeedHues("maya.chen"));
    expect(avatarSeedHues("maya.chen")).not.toEqual(avatarSeedHues("jordan.pike"));
  });

  it("derives the second and third hue from the first as the handoff specifies", () => {
    for (const seed of ["", "a", "maya.chen", "a-much-longer-username-than-usual"]) {
      const hash = fnv1a(seed);
      const [a, b, c] = avatarSeedHues(seed);
      expect(a).toBe(hash % 360);
      expect(b).toBe((a + 40 + ((hash >>> 9) % 50)) % 360);
      expect(c).toBe((a + 300 - ((hash >>> 17) % 60)) % 360);
      for (const hue of [a, b, c]) {
        expect(hue).toBeGreaterThanOrEqual(0);
        expect(hue).toBeLessThan(360);
      }
    }
  });
});

describe("avatarBackground", () => {
  it("draws a preset's hues and chroma when one is chosen", () => {
    const slate = AVATAR_PRESETS.find((preset) => preset.id === "slate");
    expect(avatarBackground("maya.chen", "slate")).toBe(avatarMesh(slate!.hues, 0.06));
  });

  it("falls back to the username's gradient for auto, an unknown style or none", () => {
    const auto = avatarMesh(avatarSeedHues("maya.chen"));
    expect(avatarBackground("maya.chen", "auto")).toBe(auto);
    expect(avatarBackground("maya.chen", "retired-preset")).toBe(auto);
    expect(avatarBackground("maya.chen", null)).toBe(auto);
  });

  it("layers four radial washes over a linear base", () => {
    const mesh = avatarMesh([10, 50, 310]);
    expect(mesh.match(/radial-gradient/g)).toHaveLength(4);
    expect(mesh).toContain("linear-gradient(135deg");
  });
});

describe("isAvatarStyle", () => {
  it("accepts auto and the presets only", () => {
    expect(isAvatarStyle("auto")).toBe(true);
    expect(isAvatarStyle("dusk")).toBe(true);
    expect(isAvatarStyle("Dusk")).toBe(false);
    expect(isAvatarStyle(undefined)).toBe(false);
  });
});
