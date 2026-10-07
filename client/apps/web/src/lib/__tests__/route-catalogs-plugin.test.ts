import { describe, expect, it } from "vitest";
import {
  requireCatalogStatement,
  routeCatalogFor,
  routeCatalogs,
} from "../../../vite/route-catalogs";

const ROUTES = "/repo/client/apps/web/src/routes";

describe("routeCatalogFor", () => {
  it("names the folder's bundle for a module anywhere inside it", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.tsx`, ROUTES)).toBe("routes/shipment");
    expect(routeCatalogFor(`${ROUTES}/shipment/_components/form/panel.tsx`, ROUTES)).toBe(
      "routes/shipment",
    );
    expect(routeCatalogFor(`${ROUTES}/admin/schema.ts`, ROUTES)).toBe("routes/admin");
  });

  it("leaves files directly in the routes folder to the shell's bundle", () => {
    expect(routeCatalogFor(`${ROUTES}/app-layout.tsx`, ROUTES)).toBeNull();
  });

  it("ignores code outside the routes folder, including a look-alike sibling", () => {
    expect(routeCatalogFor("/repo/client/apps/web/src/components/nav.tsx", ROUTES)).toBeNull();
    expect(routeCatalogFor("/repo/client/apps/web/src/routes-old/x/page.tsx", ROUTES)).toBeNull();
    expect(
      routeCatalogFor("/repo/client/packages/shared/src/routes/x/page.tsx", ROUTES),
    ).toBeNull();
  });

  it("ignores tests, stories and non-code assets", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.test.tsx`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/__tests__/helpers.ts`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/page.stories.tsx`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/styles.css`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/map.svg`, ROUTES)).toBeNull();
  });

  it("ignores another module kind of the same file", () => {
    expect(routeCatalogFor(`${ROUTES}/shipment/page.tsx?raw`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`${ROUTES}/shipment/worker.ts?worker`, ROUTES)).toBeNull();
    expect(routeCatalogFor(`\0virtual:${ROUTES}/shipment/page.tsx`, ROUTES)).toBeNull();
  });

  it("reads Windows paths the same way", () => {
    expect(
      routeCatalogFor(
        "C:\\repo\\client\\apps\\web\\src\\routes\\shipment\\page.tsx",
        "C:\\repo\\client\\apps\\web\\src\\routes\\",
      ),
    ).toBe("routes/shipment");
  });
});

describe("routeCatalogs plugin", () => {
  const plugin = routeCatalogs({ routesDir: ROUTES });
  const transform = plugin.transform as (
    code: string,
    id: string,
  ) => { code: string; map: null } | null;

  it("appends the require after the module, moving no line", () => {
    const code = 'import { useT } from "@trenova/shared/i18n/use-t";\nexport const a = 1;';
    const result = transform(code, `${ROUTES}/shipment/page.tsx`);

    expect(result?.code.startsWith(code)).toBe(true);
    expect(result?.code.slice(code.length)).toBe(requireCatalogStatement("routes/shipment"));
    expect(result?.map).toBeNull();
  });

  it("requires the bundle through the shared runtime", () => {
    expect(requireCatalogStatement("routes/shipment")).toContain(
      'from "@trenova/shared/i18n/runtime"',
    );
    expect(requireCatalogStatement("routes/shipment")).toContain(
      '__trenovaRequireCatalog("routes/shipment");',
    );
  });

  it("leaves everything else untouched", () => {
    expect(transform("export {}", "/repo/client/apps/web/src/main.tsx")).toBeNull();
  });

  it("runs before the TypeScript transform so it sees source paths", () => {
    expect(plugin.enforce).toBe("pre");
  });
});
