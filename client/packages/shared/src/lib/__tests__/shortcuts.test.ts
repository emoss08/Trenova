import { describe, expect, it } from "vitest";
import { formatShortcut, isMacPlatform } from "../shortcuts";

describe("formatShortcut", () => {
  it("uses the command glyph on a Mac and spells out Ctrl elsewhere", () => {
    expect(formatShortcut("K", true)).toBe("⌘K");
    expect(formatShortcut("K", false)).toBe("Ctrl+K");
  });

  it("keeps multi-character keys readable on both platforms", () => {
    expect(formatShortcut("Enter", true)).toBe("⌘Enter");
    expect(formatShortcut("Enter", false)).toBe("Ctrl+Enter");
  });

  it("decides from the platform when not told", () => {
    expect(formatShortcut("B")).toBe(isMacPlatform() ? "⌘B" : "Ctrl+B");
  });
});
