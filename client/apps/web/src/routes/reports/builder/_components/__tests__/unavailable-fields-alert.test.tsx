import { TestRouter } from "@trenova/shared/testing/test-router";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ReportIR } from "@/types/report";
import { buildCatalogIndex } from "../builder-state";
import { UnavailableFieldsAlert } from "../unavailable-fields-alert";
import { catalog } from "./catalog-access-fixture";

const index = buildCatalogIndex(catalog);

function renderAlert(ir: ReportIR) {
  const onChange = vi.fn();
  render(
    <TestRouter>
      <UnavailableFieldsAlert index={index} ir={ir} onChange={onChange} />
    </TestRouter>,
  );
  return onChange;
}

describe("UnavailableFieldsAlert", () => {
  it("says nothing for a report the caller may run", () => {
    const { container } = render(
      <TestRouter>
        <UnavailableFieldsAlert
          index={index}
          ir={{
            entity: "shipment",
            columns: [{ id: "c1", kind: "dimension", ref: { path: ["customer"], field: "name" } }],
          }}
          onChange={vi.fn()}
        />
      </TestRouter>,
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

  it("lists what the plan leaves out with its add-on and offers to remove only what access withholds", async () => {
    const onChange = renderAlert({
      entity: "shipment",
      columns: [
        { id: "c1", kind: "dimension", ref: { field: "proNumber" } },
        { id: "c2", kind: "measure", ref: { field: "detentionMinutes" } },
        {
          id: "c3",
          kind: "dimension",
          ref: { path: ["factoringSubmission"], field: "status" },
        },
        { id: "c4", kind: "dimension", ref: { path: ["customer"], field: "dotNumber" } },
      ],
    });

    expect(screen.getByText("3 fields in this report are unavailable to you")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Detention Minutes on Shipment is part of the Rating & Revenue Recovery add-on, which is not on your plan",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Factoring Submission on Shipment is part of the Accounting & Cash add-on, which is not on your plan",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("DOT Number on Customer needs Customers read")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Remove unavailable fields" }));

    const next = onChange.mock.calls[0][0] as ReportIR;
    expect(next.columns.map((column) => column.id)).toEqual(["c1", "c2", "c3"]);
  });

  it("offers no removal when only the plan withholds fields", () => {
    renderAlert({
      entity: "shipment",
      columns: [{ id: "c1", kind: "measure", ref: { field: "detentionMinutes" } }],
    });

    expect(screen.getByText("1 field in this report is unavailable to you")).toBeInTheDocument();
    expect(
      screen.getByText("The preview and runs are refused until your plan includes them."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove unavailable fields" })).toBeNull();
  });

  it("explains a report built on a table the plan leaves out, naming its add-on", () => {
    renderAlert({ entity: "submission", columns: [] });

    expect(screen.getByText("This report's data is not on your plan")).toBeInTheDocument();
    expect(
      screen.getByText(
        "It is built on Factoring Submissions, from the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "New report" })).toBeNull();
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
