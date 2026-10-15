import { describe, expect, it } from "vitest";
import { isScopeEnforced, scopeChoices } from "../data-scopes";

describe("scopeChoices", () => {
  it("offers what the server enforces, narrowest first", () => {
    expect(scopeChoices(["organization", "own"], "organization")).toEqual([
      { value: "own", enforced: true },
      { value: "organization", enforced: true },
    ]);
  });

  it("keeps a stored scope the server no longer enforces, marked as such", () => {
    expect(scopeChoices(["organization"], "own")).toEqual([
      { value: "own", enforced: false },
      { value: "organization", enforced: true },
    ]);
  });

  it("offers organization alone when the server sent nothing", () => {
    expect(scopeChoices(undefined, undefined)).toEqual([{ value: "organization", enforced: true }]);
  });

  it("lists a legacy organization-wide scope after organization", () => {
    expect(scopeChoices(["team", "organization"], "all")).toEqual([
      { value: "team", enforced: true },
      { value: "organization", enforced: true },
      { value: "all", enforced: false },
    ]);
  });
});

describe("isScopeEnforced", () => {
  it("says whether the stored scope is one the server enforces", () => {
    expect(isScopeEnforced(["own", "organization"], "own")).toBe(true);
    expect(isScopeEnforced(["organization"], "own")).toBe(false);
    expect(isScopeEnforced(null, "organization")).toBe(true);
  });
});
