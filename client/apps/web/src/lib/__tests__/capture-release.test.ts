import { describe, expect, it } from "vitest";
import { windowsRequirement } from "../capture-release";

describe("windowsRequirement", () => {
  it("names the Windows release a build number stands for", () => {
    expect(windowsRequirement(19045)).toEqual({ name: "Windows 10 22H2", build: 19045 });
    expect(windowsRequirement(22631)).toEqual({ name: "Windows 11 23H2", build: 22631 });
    expect(windowsRequirement(26100)).toEqual({ name: "Windows 11 24H2", build: 26100 });
  });

  it("rounds an unnamed build down to the release it belongs to", () => {
    expect(windowsRequirement(19046).name).toBe("Windows 10 22H2");
    expect(windowsRequirement(22000).name).toBe("Windows 11 21H2");
    expect(windowsRequirement(22625).name).toBe("Windows 11 22H2");
  });

  it("falls back to the bare build when it predates every named release", () => {
    expect(windowsRequirement(17763)).toEqual({ name: null, build: 17763 });
  });
});
