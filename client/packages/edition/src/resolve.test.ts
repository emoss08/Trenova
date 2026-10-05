import { describe, expect, it } from "vitest";
import defaultEdition from "./default-entry";
import { NO_PLAN, defineEdition, resolveEdition } from "./resolve";

describe("resolveEdition", () => {
  it("adds nothing for the default edition", async () => {
    const edition = resolveEdition(defaultEdition);

    expect(edition.id).toBe("self-hosted");
    expect(edition.routes).toEqual({ public: [], guest: [], protected: [], admin: [] });
    expect(edition.adminLinks).toEqual([]);
    expect(edition.navItems).toEqual([]);
    expect(edition.protectedLoaders).toEqual([]);
    expect(edition.messages).toEqual({});
    expect(edition.plan).toBe(NO_PLAN);
    await expect(edition.plan.loadRestrictions()).resolves.toEqual([]);
  });

  it("renders nothing in an empty slot and the host's fallback in the login prompt", () => {
    const { slots } = resolveEdition(undefined);
    const call = <P>(component: unknown, props: P) => (component as (props: P) => unknown)(props);

    expect(call(slots.AppBanner, {})).toBeNull();
    expect(call(slots.RootHost, {})).toBeNull();
    expect(call(slots.AuthAmbient, {})).toBeNull();
    expect(call(slots.LoginPrompt, { fallback: "Sign in" })).toBe("Sign in");
  });

  it("keeps what an edition declares and defaults the rest", () => {
    const banner = () => null;
    const edition = resolveEdition(
      defineEdition({
        id: "cloud",
        name: "Cloud",
        routes: { guest: [{ path: "/signup" }] },
        adminLinks: [{ href: "/admin/plan", title: "Plan" }],
        slots: { AppBanner: banner },
      }),
    );

    expect(edition.routes.guest).toEqual([{ path: "/signup" }]);
    expect(edition.routes.admin).toEqual([]);
    expect(edition.adminLinks).toHaveLength(1);
    expect(edition.slots.AppBanner).toBe(banner);
    expect(edition.plan).toBe(NO_PLAN);
  });
});
