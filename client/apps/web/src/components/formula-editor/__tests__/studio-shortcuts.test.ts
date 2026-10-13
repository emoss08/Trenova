import { describe, expect, it } from "vitest";
import { matchStudioShortcut, shortcutHint } from "../studio-shortcuts";

const press = (
  key: string,
  mods: Partial<{
    ctrl: boolean;
    meta: boolean;
    shift: boolean;
    alt: boolean;
    typing: boolean;
  }> = {},
) => ({
  key,
  ctrlKey: mods.ctrl ?? false,
  metaKey: mods.meta ?? false,
  shiftKey: mods.shift ?? false,
  altKey: mods.alt ?? false,
  typing: mods.typing ?? false,
});

describe("matchStudioShortcut", () => {
  it("maps the primary modifier plus a key to a studio action on both platforms", () => {
    expect(matchStudioShortcut(press("s", { ctrl: true }))).toBe("save");
    expect(matchStudioShortcut(press("s", { meta: true }))).toBe("save");
    expect(matchStudioShortcut(press("Enter", { ctrl: true }))).toBe("run");
    expect(matchStudioShortcut(press("j", { meta: true }))).toBe("nova");
    expect(matchStudioShortcut(press("J", { ctrl: true }))).toBe("nova");
    expect(matchStudioShortcut(press("1", { ctrl: true }))).toBe("preview");
    expect(matchStudioShortcut(press("2", { ctrl: true }))).toBe("scenarios");
  });

  it("ignores keys without the modifier, with extra modifiers, or unmapped keys", () => {
    expect(matchStudioShortcut(press("s"))).toBeNull();
    expect(matchStudioShortcut(press("s", { ctrl: true, shift: true }))).toBeNull();
    expect(matchStudioShortcut(press("s", { ctrl: true, alt: true }))).toBeNull();
    expect(matchStudioShortcut(press("x", { ctrl: true }))).toBeNull();
  });

  it("leaves Ctrl+K to the command palette", () => {
    expect(matchStudioShortcut(press("k", { ctrl: true }))).toBeNull();
    expect(matchStudioShortcut(press("k", { meta: true }))).toBeNull();
  });

  it("opens the reference search on a bare slash, but not while typing", () => {
    expect(matchStudioShortcut(press("/"))).toBe("search");
    expect(matchStudioShortcut(press("/", { typing: true }))).toBeNull();
    expect(matchStudioShortcut(press("/", { ctrl: true }))).toBeNull();
  });

  it("is case-insensitive for letter keys", () => {
    expect(matchStudioShortcut(press("S", { ctrl: true }))).toBe("save");
  });
});

describe("shortcutHint", () => {
  it("spells the modifier for the platform", () => {
    expect(shortcutHint("save", true)).toBe("⌘S");
    expect(shortcutHint("save", false)).toBe("Ctrl+S");
    expect(shortcutHint("run", false)).toBe("Ctrl+Enter");
    expect(shortcutHint("nova", true)).toBe("⌘J");
    expect(shortcutHint("search", false)).toBe("/");
  });
});
