import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { portalCodePrompt, portalRunStatusAttrs, portalRunTitle } from "../portal-run";

describe("portalRunTitle", () => {
  it("says what the task did and on which site", () => {
    expect(portalRunTitle(translate, { mode: "Read", siteName: "Acme DC" })).toBe(
      "Looked up on Acme DC",
    );
    expect(portalRunTitle(translate, { mode: "Act", siteName: "Acme DC" })).toBe(
      "Acted on Acme DC",
    );
  });

  it("says the site was removed when it has no name", () => {
    expect(portalRunTitle(translate, { mode: "Act", siteName: "" })).toBe(
      "Acted on a removed site",
    );
  });
});

describe("portalRunStatusAttrs", () => {
  it("puts a task that left its site in front of a person", () => {
    const strayed = portalRunStatusAttrs(translate).Strayed;
    expect(strayed.phase).toBe("attention");
    expect(strayed.text).toBe("Left the site");
  });
});

describe("portal tasks waiting on a sign-in code", () => {
  it("are waiting on the person, not running or failed", () => {
    const waiting = portalRunStatusAttrs(translate).AwaitingCode;
    expect(waiting.phase).toBe("awaiting");
    expect(waiting.text).toBe("Needs a code");
  });

  it("show how the portal asked without the browser's marker", () => {
    expect(
      portalCodePrompt("SIGN-IN CODE NEEDED: the portal texted a code to the phone ending 42."),
    ).toBe("the portal texted a code to the phone ending 42.");
    expect(portalCodePrompt("  sign-in code needed - check your email")).toBe("check your email");
    expect(portalCodePrompt("The portal asked for a code.")).toBe("The portal asked for a code.");
    expect(portalCodePrompt("SIGN-IN CODE NEEDED")).toBe("");
  });
});
