import { hotkeyLabel, hotkeyOf, keybindGroups, keybindKeys } from "@/config/keybinds.config";
import { afterEach, describe, expect, it, vi } from "vitest";

const shortcuts = vi.hoisted(() => ({ mac: false }));

vi.mock("@trenova/shared/lib/shortcuts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/shortcuts")>()),
  isMacPlatform: () => shortcuts.mac,
}));

afterEach(() => {
  shortcuts.mac = false;
});

describe("hotkeyLabel", () => {
  it("spells a keybind with Ctrl and plus signs off a Mac", () => {
    expect(hotkeyLabel("general", "user-settings")).toBe("Ctrl+Shift+S");
    expect(hotkeyLabel("general", "keyboard-shortcuts")).toBe("Ctrl+/");
  });

  it("runs the symbols together on a Mac, the way macOS menus print them", () => {
    shortcuts.mac = true;

    expect(hotkeyLabel("general", "user-settings")).toBe("⇧⌘S");
    expect(hotkeyLabel("general", "keyboard-shortcuts")).toBe("⌘/");
  });

  it("labels the same binding the app listens for", () => {
    expect(hotkeyOf("general", "user-settings")).toBe("Mod+Shift+S");
    expect(hotkeyOf("general", "keyboard-shortcuts")).toBe("Mod+/");
  });

  it("refuses a keybind that is not bound from config", () => {
    expect(() => hotkeyLabel("general", "command-palette")).toThrow(
      "No hotkey is configured for general.command-palette",
    );
  });
});

describe("keybindKeys", () => {
  it("spells the primary modifier Ctrl off a Mac and leaves the other keys as printed", () => {
    expect(keybindKeys(["Mod", "Shift", "S"], false)).toEqual(["Ctrl", "Shift", "S"]);
    expect(keybindKeys(["Mod", "Alt", "P"], false)).toEqual(["Ctrl", "Alt", "P"]);
    expect(keybindKeys(["Alt", "L"], false)).toEqual(["Alt", "L"]);
  });

  it("spells modifiers as Mac symbols, in the order macOS prints them", () => {
    expect(keybindKeys(["Mod", "Shift", "S"], true)).toEqual(["⇧", "⌘", "S"]);
    expect(keybindKeys(["Mod", "Alt", "P"], true)).toEqual(["⌥", "⌘", "P"]);
    expect(keybindKeys(["Mod", "Enter"], true)).toEqual(["⌘", "Enter"]);
    expect(keybindKeys(["G", ","], true)).toEqual(["G", ","]);
  });

  it("leaves no keybind in the config spelled with Ctrl on a Mac", () => {
    const macKeys = keybindGroups.flatMap((group) =>
      group.keybinds.flatMap((keybind) => keybindKeys(keybind.keys, true)),
    );
    expect(macKeys).not.toContain("Ctrl");
  });
});
