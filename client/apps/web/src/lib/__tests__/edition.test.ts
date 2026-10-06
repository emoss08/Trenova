import type { SidebarLink } from "@/components/sidebar-nav";
import type { NavGroup, NavItem, NavModule } from "@/config/navigation.types";
import { mergeAdminLinks, mergeNavItems } from "@/lib/edition";
import { Home01Icon } from "@trenova/shared/components/icons";
import { describe, expect, it } from "vitest";

const hostLinks: SidebarLink[] = [
  { href: "/admin/a", title: "A" },
  { href: "/admin/b", title: "B" },
];

describe("mergeAdminLinks", () => {
  it("returns the host's links untouched when the edition adds none", () => {
    const merged = mergeAdminLinks(hostLinks, []);

    expect(merged).toEqual(hostLinks);
    expect(merged).not.toBe(hostLinks);
  });

  it("places a link after the one it names, without carrying the anchor along", () => {
    const merged = mergeAdminLinks(hostLinks, [
      { href: "/admin/plan", title: "Plan", platformMode: "cloud", after: "/admin/a" },
    ]);

    expect(merged.map((link) => link.href)).toEqual(["/admin/a", "/admin/plan", "/admin/b"]);
    expect(merged[1]).toEqual({ href: "/admin/plan", title: "Plan", platformMode: "cloud" });
  });

  it("appends a link with no anchor, or one naming a link the host does not have", () => {
    const merged = mergeAdminLinks(hostLinks, [
      { href: "/admin/x", title: "X" },
      { href: "/admin/y", title: "Y", after: "/admin/missing" },
    ]);

    expect(merged.map((link) => link.href)).toEqual([
      "/admin/a",
      "/admin/b",
      "/admin/x",
      "/admin/y",
    ]);
  });

  it("keeps two additions anchored to one link in declaration order", () => {
    const merged = mergeAdminLinks(hostLinks, [
      { href: "/admin/1", title: "1", after: "/admin/a" },
      { href: "/admin/2", title: "2", after: "/admin/1" },
    ]);

    expect(merged.map((link) => link.href)).toEqual([
      "/admin/a",
      "/admin/1",
      "/admin/2",
      "/admin/b",
    ]);
  });
});

const first: NavItem = { id: "first", label: "First", path: "/m/first" };
const group: NavGroup = {
  id: "config",
  label: "Configuration",
  items: [{ id: "g1", label: "G1", path: "/m/g1" }],
};
const hostModule: NavModule = {
  id: "admin",
  label: "Admin",
  icon: Home01Icon,
  basePath: "/m",
  navigation: [first, group],
};
const otherModule: NavModule = { ...hostModule, id: "home", basePath: "/", navigation: [] };

describe("mergeNavItems", () => {
  it("returns the host's modules untouched when the edition adds none", () => {
    expect(mergeNavItems([hostModule], [])).toEqual([hostModule]);
  });

  it("adds an item to a module's top level after the sibling it names", () => {
    const [merged] = mergeNavItems(
      [hostModule],
      [{ moduleId: "admin", after: "first", item: { id: "new", label: "New", path: "/m/new" } }],
    );

    expect(merged.navigation.map((entry) => entry.id)).toEqual(["first", "new", "config"]);
    expect(hostModule.navigation).toHaveLength(2);
  });

  it("adds an item into a group, and to the top level when the group is unknown", () => {
    const [merged] = mergeNavItems(
      [hostModule],
      [
        { moduleId: "admin", groupId: "config", item: { id: "g2", label: "G2", path: "/m/g2" } },
        { moduleId: "admin", groupId: "nope", item: { id: "loose", label: "L", path: "/m/l" } },
      ],
    );

    const mergedGroup = merged.navigation.find((entry) => entry.id === "config") as NavGroup;
    expect(mergedGroup.items.map((item) => item.id)).toEqual(["g1", "g2"]);
    expect(group.items).toHaveLength(1);
    expect(merged.navigation.map((entry) => entry.id)).toEqual(["first", "config", "loose"]);
  });

  it("drops an item for a module the host does not have and leaves other modules alone", () => {
    const merged = mergeNavItems(
      [hostModule, otherModule],
      [{ moduleId: "missing", item: { id: "x", label: "X", path: "/x" } }],
    );

    expect(merged).toEqual([hostModule, otherModule]);
    expect(merged[1]).toBe(otherModule);
  });
});
