import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { portalRunStatusAttrs, portalRunTitle } from "../portal-run";

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
