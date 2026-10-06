import { describe, expect, it } from "vitest";
import {
  businessHoursLabel,
  capabilityOptions,
  clockLabel,
  isLocked,
  limitHigh,
  limitShare,
  withToolMode,
  type AgentCapabilities,
  type AgentCapabilityTool,
} from "../agent-capabilities";

const read: AgentCapabilityTool = {
  key: "list_invoices",
  label: "List invoices",
  write: false,
  mode: "Allowed",
  allowedModes: ["Allowed", "Off"],
  lockReason: null,
};

const post: AgentCapabilityTool = {
  key: "post_invoices",
  label: "Post invoices",
  write: true,
  mode: "AskFirst",
  allowedModes: ["AskFirst", "Off"],
  lockReason: "Always asks · this can't be undone",
};

const assign: AgentCapabilityTool = {
  key: "assign_biller",
  label: "Assign biller",
  write: true,
  mode: "Allowed",
  allowedModes: ["Allowed", "AskFirst", "Off"],
  lockReason: null,
};

describe("capabilityOptions", () => {
  it("never offers Ask first for a read", () => {
    expect(capabilityOptions(read, true).map((option) => option.mode)).toEqual(["Allowed", "Off"]);
  });

  it("draws every mode of a write and selects the current one", () => {
    const options = capabilityOptions(assign, true);
    expect(options.map((option) => option.mode)).toEqual(["Allowed", "AskFirst", "Off"]);
    expect(options.filter((option) => option.selected).map((option) => option.mode)).toEqual([
      "Allowed",
    ]);
    expect(options.every((option) => !option.disabled)).toBe(true);
  });

  it("disables what a lock rules out but still shows where the row stands", () => {
    const options = capabilityOptions(post, true);
    expect(isLocked(post)).toBe(true);
    expect(options.find((option) => option.mode === "Allowed")?.disabled).toBe(true);
    expect(options.find((option) => option.mode === "AskFirst")).toMatchObject({
      selected: true,
      disabled: false,
    });
  });

  it("disables every option for a reader who may not edit", () => {
    expect(capabilityOptions(assign, false).every((option) => option.disabled)).toBe(true);
  });
});

describe("withToolMode", () => {
  it("moves only the row that changed", () => {
    const caps = { readTools: [read], writeTools: [post, assign] } as unknown as AgentCapabilities;
    const next = withToolMode(caps, "assign_biller", "Off");
    expect(next.writeTools.map((tool) => tool.mode)).toEqual(["AskFirst", "Off"]);
    expect(next.readTools[0]).toBe(read);
    expect(caps.writeTools[1].mode).toBe("Allowed");
  });
});

describe("limits", () => {
  it("turns amber past 85 percent", () => {
    expect(limitHigh(460, 500)).toBe(true);
    expect(limitHigh(425, 500)).toBe(false);
    expect(limitHigh(143, 200)).toBe(false);
    expect(limitHigh(10, 0)).toBe(false);
  });

  it("caps the bar at full and reads no limit as empty", () => {
    expect(limitShare(143, 200)).toBeCloseTo(0.715);
    expect(limitShare(600, 500)).toBe(1);
    expect(limitShare(5, 0)).toBe(0);
  });
});

describe("business hours", () => {
  it("writes minutes after midnight as a clock time", () => {
    expect(clockLabel(420)).toBe("7 AM");
    expect(clockLabel(1080)).toBe("6 PM");
    expect(clockLabel(750)).toBe("12:30 PM");
    expect(clockLabel(0)).toBe("12 AM");
  });

  it("names the window and its zone", () => {
    expect(businessHoursLabel(420, 1080, "America/Chicago")).toBe("7 AM – 6 PM Central");
    expect(businessHoursLabel(420, 1080, "Not/AZone")).toBe("7 AM – 6 PM Not/AZone");
  });
});
