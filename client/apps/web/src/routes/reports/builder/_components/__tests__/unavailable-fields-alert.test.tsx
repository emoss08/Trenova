import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { ReportIR } from "@/types/report";
import { buildCatalogIndex } from "../builder-state";
import { UnavailableFieldsAlert } from "../unavailable-fields-alert";
import { catalog } from "./catalog-access-fixture";

const index = buildCatalogIndex(catalog);

function renderAlert(ir: ReportIR) {
  const onChange = vi.fn();
  render(
    <MemoryRouter>
      <UnavailableFieldsAlert index={index} ir={ir} onChange={onChange} />
    </MemoryRouter>,
  );
  return onChange;
}

describe("UnavailableFieldsAlert", () => {
  it("says nothing for a report the caller may run", () => {
    const { container } = render(
      <MemoryRouter>
        <UnavailableFieldsAlert
          index={index}
          ir={{
            entity: "shipment",
            columns: [{ id: "c1", kind: "dimension", ref: { path: ["customer"], field: "name" } }],
          }}
          onChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it("names each unavailable field and what it needs, and removes them on request", async () => {
    const onChange = renderAlert({
      entity: "shipment",
      columns: [
        { id: "c1", kind: "dimension", ref: { field: "proNumber" } },
        { id: "c2", kind: "dimension", ref: { path: ["customer"], field: "dotNumber" } },
        { id: "c3", kind: "dimension", ref: { field: "marginFloorPercent" } },
      ],
      sort: [{ columnId: "c2", direction: "asc" }],
    });

    expect(screen.getByText("2 fields in this report are unavailable to you")).toBeInTheDocument();
    expect(screen.getByText("DOT Number on Customer needs Customers read")).toBeInTheDocument();
    expect(
      screen.getByText("Margin Floor on Shipment needs the Margins field group"),
    ).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Remove unavailable fields" }));

    expect(onChange).toHaveBeenCalledTimes(1);
    const next = onChange.mock.calls[0][0] as ReportIR;
    expect(next.columns.map((column) => column.id)).toEqual(["c1"]);
    expect(next.sort ?? []).toEqual([]);
  });

  it("explains a report built on a kind the caller cannot read", () => {
    renderAlert({ entity: "customer", columns: [] });

    expect(screen.getByText("You cannot open this report's data")).toBeInTheDocument();
    expect(
      screen.getByText(
        "It is built on Customers, which your role cannot read. Ask an administrator for access, or start a new report.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "New report" })).toHaveAttribute(
      "href",
      "/reports/builder",
    );
  });
});
