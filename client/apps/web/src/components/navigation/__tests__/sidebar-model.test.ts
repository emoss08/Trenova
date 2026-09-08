import type { SidebarLink } from "@/components/sidebar-nav";
import { navigationConfig } from "@/config/navigation.config";
import type { NavModule } from "@/config/navigation.types";
import { HomeIcon } from "lucide-react";
import { describe, expect, it } from "vitest";
import {
  buildModuleView,
  collectNavPaths,
  findModuleForPath,
  groupModulesByDomain,
  moduleAttention,
  moduleDisplayLabel,
  pageAttention,
} from "../sidebar-model";

const modules = navigationConfig.modules;

function moduleById(id: string): NavModule {
  const found = modules.find((module) => module.id === id);
  if (!found) throw new Error(`module ${id} is not registered`);
  return found;
}

describe("moduleDisplayLabel", () => {
  it("prefers the short label and falls back to the full one", () => {
    expect(moduleDisplayLabel(moduleById("hr"))).toBe("People");
    expect(moduleDisplayLabel(moduleById("carrier-settlements"))).toBe("Carrier Settlements");
  });
});

describe("findModuleForPath", () => {
  it("matches home only on the root path", () => {
    expect(findModuleForPath(modules, "/")?.id).toBe("home");
    expect(findModuleForPath(modules, "/hr")?.id).not.toBe("home");
  });

  it("resolves a module whose pages live under a prefix other than its base path", () => {
    expect(findModuleForPath(modules, "/shipment-management/shipments")?.id).toBe("shipment");
  });

  it("owns every prefix a module declares", () => {
    expect(findModuleForPath(modules, "/admin/api-keys")?.id).toBe("admin");
    expect(findModuleForPath(modules, "/organization/email-logs/")?.id).toBe("admin");
  });

  it("does not match a sibling that merely shares a leading string", () => {
    expect(findModuleForPath(modules, "/hrx/anything")).toBeNull();
    expect(findModuleForPath(modules, "/hr/workers/wrk_01")?.id).toBe("hr");
  });

  it("returns null for a path no module claims", () => {
    expect(findModuleForPath(modules, "/profile")).toBeNull();
  });
});

describe("buildModuleView", () => {
  it("splits working pages from the configuration catalogue", () => {
    const view = buildModuleView(moduleById("hr"), []);
    expect(view.sections).toHaveLength(1);
    expect(view.sections[0].label).toBeNull();
    expect(view.sections[0].items.map((item) => item.id)).toEqual(["workers", "my-team"]);
    expect(view.configuration.map((item) => item.id)).toContain("credential-types");
    expect(view.configuration.map((item) => item.id)).not.toContain("workers");
    expect(view.landingPath).toBe("/hr/workers");
  });

  it("keeps a module's non-configuration groups as labelled sections in order", () => {
    const view = buildModuleView(moduleById("accounting"), []);
    expect(view.sections.map((section) => section.label)).toEqual([
      null,
      "Reports",
      "Accounts Receivable",
      "Bank Reconciliation",
    ]);
    expect(view.configuration.map((item) => item.id)).toEqual(["account-types", "fiscal-years"]);
  });

  it("starts a new unlabelled section when plain pages follow a group", () => {
    const module: NavModule = {
      id: "reports",
      label: "Reports",
      icon: HomeIcon,
      basePath: "/reports",
      navigation: [
        { id: "a", label: "A", path: "/reports/a" },
        { id: "g", label: "Group", items: [{ id: "b", label: "B", path: "/reports/b" }] },
        { id: "c", label: "C", path: "/reports/c" },
      ],
    };
    const view = buildModuleView(module, []);
    expect(view.sections.map((section) => section.items.map((item) => item.id))).toEqual([
      ["a"],
      ["b"],
      ["c"],
    ]);
  });

  it("lands on the module base path when it has no pages", () => {
    const module: NavModule = {
      id: "reports",
      label: "Reports",
      icon: HomeIcon,
      basePath: "/reports",
      navigation: [],
    };
    expect(buildModuleView(module, []).landingPath).toBe("/reports");
  });

  it("lands on the first configuration page when that is all a module has", () => {
    const module: NavModule = {
      id: "reports",
      label: "Reports",
      icon: HomeIcon,
      basePath: "/reports",
      navigation: [
        {
          id: "cfg",
          label: "Configuration Files",
          kind: "configuration",
          items: [{ id: "x", label: "X", path: "/reports/x" }],
        },
      ],
    };
    expect(buildModuleView(module, []).landingPath).toBe("/reports/x");
  });

  it("builds the settings module from the admin links, one section per group", () => {
    const links: SidebarLink[] = [
      { href: "/admin/users/", title: "Users", group: "Organization" },
      { href: "/admin/api-keys", title: "API Keys", group: "Data & Integrations" },
      { href: "/admin/hold-reasons/", title: "Hold Reasons", group: "Organization" },
      { href: "/admin/loose", title: "Loose", includeBetaTag: true },
    ];
    const view = buildModuleView(moduleById("admin"), links);
    expect(view.sections.map((section) => section.label)).toEqual([
      "Organization",
      "Data & Integrations",
      "Other",
    ]);
    expect(view.sections[0].items.map((item) => item.path)).toEqual([
      "/admin/users/",
      "/admin/hold-reasons/",
    ]);
    expect(view.sections[2].items[0].includeBetaTag).toBe(true);
    expect(view.configuration).toEqual([]);
    expect(view.landingPath).toBe("/admin/organization-settings");
  });
});

describe("groupModulesByDomain", () => {
  it("keeps the configured domain order and drops domains with nothing to show", () => {
    const domains = groupModulesByDomain(modules.filter((module) => module.id !== "home"));
    expect(domains.map((domain) => domain.id)).toEqual([
      "operations",
      "people",
      "finance",
      "admin",
    ]);
    expect(domains[0].modules.map((module) => module.id)).toEqual([
      "shipment",
      "dispatch",
      "fleet",
      "edi",
    ]);
    expect(domains[1].modules.map((module) => module.id)).toEqual(["hr", "payroll"]);
  });

  it("collects a module no domain claims into a trailing group", () => {
    const stray: NavModule = {
      id: "organization",
      label: "Stray",
      icon: HomeIcon,
      basePath: "/stray",
      navigation: [],
    };
    const domains = groupModulesByDomain([moduleById("billing"), stray]);
    expect(domains.at(-1)?.modules).toEqual([stray]);
    expect(domains.at(-1)?.id).toBe("other");
  });
});

describe("collectNavPaths", () => {
  it("includes every page, configuration item and admin link exactly once", () => {
    const links: SidebarLink[] = [{ href: "/admin/users/", title: "Users", group: "Organization" }];
    const views = [
      buildModuleView(moduleById("hr"), []),
      buildModuleView(moduleById("admin"), links),
    ];
    const paths = collectNavPaths(views);
    expect(paths).toContain("/hr/workers");
    expect(paths).toContain("/hr/credential-types");
    expect(paths).toContain("/admin/users/");
    expect(new Set(paths).size).toBe(paths.length);
  });
});

describe("moduleAttention", () => {
  const summary = {
    billingQueue: 4,
    pendingApprovals: 0,
    reconciliationExceptions: 2,
    serviceFailures: 7,
    ediAttention: null,
  };

  it("sums the counts that land in a module and keeps the strongest tone", () => {
    const attention = moduleAttention(modules, summary, [
      "billingQueue",
      "reconciliationExceptions",
      "serviceFailures",
      "ediAttention",
    ]);
    expect(attention.get("billing")).toEqual({ count: 6, tone: "destructive" });
    expect(attention.get("shipment")).toEqual({ count: 7, tone: "warning" });
    expect(attention.has("edi")).toBe(false);
  });

  it("only counts the metrics the person chose to watch", () => {
    const attention = moduleAttention(modules, summary, ["billingQueue"]);
    expect(attention.get("billing")).toEqual({ count: 4, tone: "default" });
    expect(attention.has("shipment")).toBe(false);
  });

  it("ignores zero counts and metrics the summary does not carry", () => {
    const attention = moduleAttention(modules, summary, ["pendingApprovals", "ediAttention"]);
    expect(attention.size).toBe(0);
  });
});

describe("pageAttention", () => {
  const summary = { billingQueue: 4, reconciliationExceptions: 0, serviceFailures: 7 };

  it("keys the watched counts by the page they belong to", () => {
    const attention = pageAttention(summary, ["billingQueue", "serviceFailures"]);
    expect(attention.get("/billing/queue")).toEqual({ count: 4, tone: "default" });
    expect(attention.get("/shipment-management/service-failures")).toEqual({
      count: 7,
      tone: "warning",
    });
  });

  it("leaves out unwatched metrics, zero counts and unknown keys", () => {
    const attention = pageAttention(summary, ["reconciliationExceptions", "nope"]);
    expect(attention.size).toBe(0);
    expect(pageAttention(summary, ["serviceFailures"]).has("/billing/queue")).toBe(false);
  });

  it("returns an empty map before the summary has loaded", () => {
    expect(pageAttention(undefined, ["billingQueue"]).size).toBe(0);
  });
});

describe("module ownership of its own pages", () => {
  it("resolves every page of every module back to that module", () => {
    const misfiled: string[] = [];
    for (const module of modules) {
      const view = buildModuleView(module, []);
      const pages = [...view.sections.flatMap((section) => section.items), ...view.configuration];
      for (const page of pages) {
        const owner = findModuleForPath(modules, page.path);
        if (owner?.id !== module.id) {
          misfiled.push(`${page.path} -> ${owner?.id ?? "nobody"} (expected ${module.id})`);
        }
      }
    }
    expect(misfiled).toEqual([]);
  });

  it("keeps a module's deeper pages inside the module", () => {
    expect(findModuleForPath(modules, "/edi/designer")?.id).toBe("edi");
    expect(findModuleForPath(modules, "/edi/partners/prt_01")?.id).toBe("edi");
  });
});
