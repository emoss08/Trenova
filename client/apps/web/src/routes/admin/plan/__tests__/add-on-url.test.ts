import { describe, expect, it } from "vitest";
import { addOnUrl } from "../_components/add-on-url";

describe("addOnUrl", () => {
  it("asks the account's plan page to switch the add-on on", () => {
    expect(addOnUrl("https://console.trenova.cloud/instances/1/plan", "DSP")).toBe(
      "https://console.trenova.cloud/instances/1/plan?add=DSP",
    );
  });

  it("keeps a query the plan page already carries and replaces an earlier add", () => {
    expect(
      addOnUrl("https://console.trenova.cloud/instances/1/plan?tab=addons&add=ACC", "DSP"),
    ).toBe("https://console.trenova.cloud/instances/1/plan?tab=addons&add=DSP");
  });

  it("draws no link for an address it cannot read", () => {
    expect(addOnUrl("not a url", "DSP")).toBeNull();
  });
});
