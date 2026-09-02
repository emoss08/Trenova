import type { ImportTemplatesResponse } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { formulaTemplateRoutes, importLandingRoute } from "../formula-template-routes";

describe("formulaTemplateRoutes", () => {
  it("builds the studio routes from one base", () => {
    expect(formulaTemplateRoutes.list).toBe("/billing/configuration-files/formula-templates");
    expect(formulaTemplateRoutes.new).toBe("/billing/configuration-files/formula-templates/new");
    expect(formulaTemplateRoutes.edit("ft_1")).toBe(
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
  });
});

describe("importLandingRoute", () => {
  it("opens the studio when exactly one template was imported", () => {
    const response = { created: [{ id: "ft_9" }], renamed: null } as ImportTemplatesResponse;
    expect(importLandingRoute(response)).toBe(formulaTemplateRoutes.edit("ft_9"));
  });

  it("returns to the list for several imports or an import without an id", () => {
    expect(
      importLandingRoute({ created: [{ id: "a" }, { id: "b" }] } as ImportTemplatesResponse),
    ).toBe(formulaTemplateRoutes.list);
    expect(importLandingRoute({ created: [{}] } as ImportTemplatesResponse)).toBe(
      formulaTemplateRoutes.list,
    );
  });
});
