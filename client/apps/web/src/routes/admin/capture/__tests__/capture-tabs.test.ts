import { captureSettingFields, documentControlSchema } from "@/types/document-control";
import { describe, expect, it } from "vitest";
import { captureTabs } from "../capture-tabs";

const ALL = { settings: true, profiles: true, computers: true };

describe("scanning and printing tabs", () => {
  it("puts every capture setting the record carries on the settings tab, and nowhere else", () => {
    const capture = Object.keys(documentControlSchema.shape).filter(
      (key) => key.startsWith("capture") || key === "enableCapture",
    );
    const [settings, ...others] = captureTabs(ALL);

    expect(settings!.value).toBe("settings");
    expect([...settings!.fields].sort()).toEqual(capture.sort());
    expect([...captureSettingFields].sort()).toEqual(capture.sort());
    expect(others.flatMap((tab) => tab.fields)).toEqual([]);
  });

  it("offers settings, scan profiles and computers, in that order", () => {
    expect(captureTabs(ALL).map((tab) => [tab.value, tab.label])).toEqual([
      ["settings", "Settings"],
      ["profiles", "Scan profiles"],
      ["computers", "Computers"],
    ]);
  });

  it.each([
    [{ settings: false, profiles: true, computers: true }, ["profiles", "computers"]],
    [{ settings: true, profiles: true, computers: false }, ["settings", "profiles"]],
    [{ settings: false, profiles: true, computers: false }, ["profiles"]],
    [{ settings: false, profiles: false, computers: false }, []],
  ])("shows only the tabs that may be read (%o)", (access, expected) => {
    expect(captureTabs(access).map((tab) => tab.value)).toEqual(expected);
  });
});
