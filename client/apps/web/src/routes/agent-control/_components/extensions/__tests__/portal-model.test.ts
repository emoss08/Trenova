import type { PortalSite } from "@/types/agent-extension";
import { describe, expect, it } from "vitest";
import { normalizeDomains, siteFormDefaults, siteRequestOf } from "../portal-model";

function site(overrides: Partial<PortalSite> = {}): PortalSite {
  return {
    id: "apst_01J8ZK3M4N5P6Q7R8S9T0V1W2X",
    name: "Acme DC",
    purpose: "dock appointments",
    domains: ["portal.acme.com"],
    startUrl: "https://portal.acme.com/login",
    browserProfileId: "",
    allowActions: true,
    enabled: true,
    hasSignIn: true,
    version: 3,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    ...overrides,
  };
}

describe("normalizeDomains", () => {
  it("keeps only the hostname of each address, once", () => {
    expect(
      normalizeDomains([
        " https://Portal.Acme.com/login?next=/ ",
        "portal.acme.com",
        "sso.acme.com:8443",
        "https://user:pass@dock.acme.com/path",
        "*.dockhub.io",
        "acme.com.",
        "   ",
      ]),
    ).toEqual(["portal.acme.com", "sso.acme.com", "dock.acme.com", "*.dockhub.io", "acme.com"]);
  });
});

describe("siteFormDefaults", () => {
  it("never puts a stored sign-in back in the form", () => {
    expect(siteFormDefaults(site())).toEqual({
      name: "Acme DC",
      purpose: "dock appointments",
      domains: ["portal.acme.com"],
      startUrl: "https://portal.acme.com/login",
      username: "",
      password: "",
      clearSignIn: false,
      browserProfileId: "",
      allowActions: true,
      enabled: true,
    });
  });

  it("starts a new site turned on and read-only", () => {
    const defaults = siteFormDefaults(null);
    expect(defaults.enabled).toBe(true);
    expect(defaults.allowActions).toBe(false);
    expect(defaults.domains).toEqual([]);
  });
});

describe("siteRequestOf", () => {
  it("sends the trimmed values with the site's version", () => {
    expect(
      siteRequestOf(
        {
          ...siteFormDefaults(site()),
          name: "  Acme DC  ",
          domains: ["https://portal.acme.com/login"],
          username: " dispatch@trenova.test ",
          password: " s3cret ",
        },
        site(),
      ),
    ).toEqual({
      name: "Acme DC",
      purpose: "dock appointments",
      domains: ["portal.acme.com"],
      startUrl: "https://portal.acme.com/login",
      username: "dispatch@trenova.test",
      password: " s3cret ",
      clearSignIn: false,
      browserProfileId: "",
      allowActions: true,
      enabled: true,
      version: 3,
    });
  });

  it("sends neither part of a sign-in being cleared", () => {
    const request = siteRequestOf(
      { ...siteFormDefaults(site()), username: "someone", password: "typed", clearSignIn: true },
      site(),
    );
    expect(request.username).toBe("");
    expect(request.password).toBe("");
    expect(request.clearSignIn).toBe(true);
  });

  it("sends version 0 for a new site", () => {
    expect(siteRequestOf(siteFormDefaults(null), null).version).toBe(0);
  });
});
