import type { SidebarModuleView, SidebarPageItem } from "@/components/navigation/sidebar-model";
import { describe, expect, it } from "vitest";
import { sectionIndexPanels } from "../section-index";

const Icon = () => null;

function page(id: string, groupId: string | null, extra: Partial<SidebarPageItem> = {}) {
  return { id, label: id, path: `/billing/${id}`, groupId, ...extra };
}

const view: SidebarModuleView = {
  module: {
    id: "billing",
    label: "Billing management",
    icon: Icon,
    basePath: "/billing",
    navigation: [],
  },
  sections: [
    { id: "billing:queue", label: null, items: [page("queue", null), page("invoices", null)] },
    {
      id: "factoring",
      label: "Factoring",
      items: [
        page("submissions", "factoring"),
        page("remittances", "factoring", { disabled: true }),
      ],
    },
    { id: "empty", label: "Empty", items: [page("soon", "empty", { disabled: true })] },
  ],
  configuration: [
    page("charge-types", "billing-config-group"),
    page("rate-zones", "billing-config-group"),
  ],
  landingPath: "/billing/queue",
};

describe("sectionIndexPanels", () => {
  it("lists a module's groups in sidebar order with its configuration last", () => {
    expect(
      sectionIndexPanels(view, null).map((panel) => [
        panel.label,
        panel.configuration,
        panel.items.map((item) => item.id),
      ]),
    ).toEqual([
      [null, false, ["queue", "invoices"]],
      ["Factoring", false, ["submissions"]],
      [null, true, ["charge-types", "rate-zones"]],
    ]);
  });

  it("lists only the group's own pages for a group section, configuration groups included", () => {
    expect(sectionIndexPanels(view, "billing-config-group")).toEqual([
      {
        id: "billing-config-group",
        label: null,
        configuration: false,
        items: [
          page("charge-types", "billing-config-group"),
          page("rate-zones", "billing-config-group"),
        ],
      },
    ]);
    expect(sectionIndexPanels(view, "factoring")[0].items.map((item) => item.id)).toEqual([
      "submissions",
    ]);
  });

  it("is empty when the person may open nothing there", () => {
    expect(sectionIndexPanels(view, "empty")).toEqual([]);
    expect(sectionIndexPanels({ ...view, sections: [], configuration: [] }, null)).toEqual([]);
  });
});
