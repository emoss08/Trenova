import { describe, expect, it } from "vitest";
import { chunkGroups, packageTest } from "../../../vite/chunk-groups";

const PNPM = "/repo/client/node_modules/.pnpm";

function pnpmModule(folder: string, pkg: string, file: string): string {
  return `${PNPM}/${folder}/node_modules/${pkg}/${file}`;
}

// Rolldown places a module in the first group whose test matches it; a group tagged
// `$initial` only takes modules the entry imports statically.
function groupOf(id: string, { startup = false } = {}): string | null {
  for (const group of chunkGroups) {
    if (group.tags?.includes("$initial") && !startup) continue;
    const { test } = group;
    const matched = test instanceof RegExp ? test.test(id) : typeof test === "function" && test(id);
    if (matched) return typeof group.name === "string" ? group.name : null;
  }
  return null;
}

describe("packageTest", () => {
  it("matches a package by its own folder in pnpm's layout", () => {
    const test = packageTest("date-fns");
    expect(test.test(pnpmModule("date-fns@4.4.0", "date-fns", "format.js"))).toBe(true);
    expect(test.test("/repo/client/node_modules/date-fns/index.js")).toBe(true);
  });

  it("does not match a package that only names it as a peer", () => {
    const baseUi = pnpmModule(
      "@base-ui+react@1.8.0_@date-fns+tz@1.4.1_date-fns@4.4.0_react@19.2.8",
      "@base-ui/react",
      "esm/button/Button.js",
    );
    expect(packageTest("date-fns").test(baseUi)).toBe(false);
    expect(packageTest("@date-fns/*").test(baseUi)).toBe(false);
  });

  it("does not match app code whose path contains the package's name", () => {
    expect(packageTest("motion").test("/repo/client/apps/web/src/stores/motion-store.ts")).toBe(
      false,
    );
    expect(packageTest("zod").test("/repo/client/packages/shared/src/lib/zod-helpers.ts")).toBe(
      false,
    );
  });

  it("does not match a package whose name merely starts with the given one", () => {
    expect(packageTest("react").test(pnpmModule("react-dom@19.2.8", "react-dom", "index.js"))).toBe(
      false,
    );
    expect(packageTest("motion").test(pnpmModule("motion-dom@12.0.0", "motion-dom", "a.js"))).toBe(
      false,
    );
  });

  it("expands * within one path segment", () => {
    const tiptap = packageTest("@tiptap/*", "prosemirror-*");
    expect(tiptap.test(pnpmModule("@tiptap+core@3.0.0", "@tiptap/core", "dist/index.js"))).toBe(
      true,
    );
    expect(tiptap.test(pnpmModule("prosemirror-view@1.0.0", "prosemirror-view", "dist/a.js"))).toBe(
      true,
    );
    expect(tiptap.test(pnpmModule("@tiptapx+core@1.0.0", "@tiptapx/core", "a.js"))).toBe(false);
  });

  it("matches a subpath of a package and nothing else in it", () => {
    const flags = packageTest("react-phone-number-input/flags");
    expect(
      flags.test(
        pnpmModule("react-phone-number-input@3.4.18", "react-phone-number-input", "flags/index.js"),
      ),
    ).toBe(true);
    expect(
      flags.test(
        pnpmModule("react-phone-number-input@3.4.18", "react-phone-number-input", "core/index.js"),
      ),
    ).toBe(false);
  });

  it("reads Windows paths the same way", () => {
    expect(
      packageTest("@tanstack/react-table").test(
        "C:\\repo\\client\\node_modules\\.pnpm\\@tanstack+react-table@9.2.4\\node_modules\\@tanstack\\react-table\\dist\\index.js",
      ),
    ).toBe(true);
  });
});

describe("chunkGroups", () => {
  it("leaves Base UI, icons and date-fns to per-route chunks", () => {
    expect(
      groupOf(pnpmModule("@base-ui+react@1.8.0_date-fns@4.4.0", "@base-ui/react", "esm/index.js")),
    ).toBeNull();
    expect(
      groupOf(pnpmModule("@untitledui+icons@0.0.20", "@untitledui/icons", "dist/a.js")),
    ).toBeNull();
    expect(groupOf(pnpmModule("date-fns@4.4.0", "date-fns", "format.js"))).toBeNull();
  });

  it("keeps the app's own code out of every group", () => {
    expect(groupOf("/repo/client/apps/web/src/lib/route-prefetch.ts")).toBeNull();
    expect(groupOf("/repo/client/apps/web/src/stores/motion-preference-store.ts")).toBeNull();
    expect(groupOf("/repo/client/packages/shared/src/lib/utils.ts")).toBeNull();
  });

  it("gathers what the entry needs at startup into one chunk per source", () => {
    const startup = { startup: true };
    expect(
      groupOf(
        pnpmModule("@base-ui+react@1.8.0_date-fns@4.4.0", "@base-ui/react", "esm/index.js"),
        startup,
      ),
    ).toBe("initial-vendor");
    expect(groupOf("/repo/client/packages/graphql/src/fragment-data.ts", startup)).toBe("initial");
    expect(groupOf("/repo/client/packages/shared/src/lib/utils.ts", startup)).toBe(
      "initial-shared",
    );
    expect(groupOf("/repo/client/packages/edition/src/default-entry.ts", startup)).toBe(
      "initial-shared",
    );
    expect(groupOf("/repo/client/apps/web/src/router.tsx", startup)).toBe("initial");
  });

  it("puts every startup stylesheet, the app's and its dependencies', in one file", () => {
    const startup = { startup: true };
    expect(groupOf("/repo/client/packages/shared/src/styles/app.css", startup)).toBe(
      "initial-styles",
    );
    expect(groupOf("/repo/client/apps/web/src/components/desk-chat/desk-frame.css", startup)).toBe(
      "initial-styles",
    );
    expect(
      groupOf(
        pnpmModule(
          "react-lazy-load-image-component@1.6.3",
          "react-lazy-load-image-component",
          "src/effects/blur.css",
        ),
        startup,
      ),
    ).toBe("initial-styles");
  });

  it("leaves modules a barrel re-exports to the routes that use them, even at startup", () => {
    const startup = { startup: true };
    expect(
      groupOf(
        pnpmModule("@untitledui+icons@0.0.20", "@untitledui/icons", "dist/Truck01.mjs"),
        startup,
      ),
    ).toBeNull();
    expect(
      groupOf("/repo/client/packages/graphql/src/generated/documents/shipment/table.ts", startup),
    ).toBeNull();
    expect(groupOf("/repo/client/packages/graphql/src/fragment-data.ts", startup)).toBe("initial");
  });

  it("leaves a startup library that has its own chunk in that chunk", () => {
    expect(
      groupOf(pnpmModule("react-dom@19.2.8", "react-dom", "client.js"), { startup: true }),
    ).toBe("react-vendor");
  });

  it("leaves the HTML entry and Vite's virtual modules with the entry chunk", () => {
    const startup = { startup: true };
    expect(groupOf("/repo/client/apps/web/index.html", startup)).toBeNull();
    expect(groupOf("\0vite/modulepreload-polyfill.js", startup)).toBeNull();
  });

  it("puts TanStack Store with Query, not with the lazily loaded table", () => {
    expect(groupOf(pnpmModule("@tanstack+store@0.8.0", "@tanstack/store", "dist/index.js"))).toBe(
      "tanstack-query",
    );
    expect(
      groupOf(pnpmModule("@tanstack+table-core@9.2.4", "@tanstack/table-core", "dist/index.js")),
    ).toBe("tanstack-table");
  });

  it("puts the lazily imported flag barrel with the flags, ahead of the phone input", () => {
    expect(
      groupOf(
        pnpmModule("react-phone-number-input@3.4.18", "react-phone-number-input", "flags/index.js"),
      ),
    ).toBe("country-flags");
    expect(
      groupOf(
        pnpmModule("react-phone-number-input@3.4.18", "react-phone-number-input", "min/index.js"),
      ),
    ).toBe("phone-utils");
  });
});
