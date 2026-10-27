import type { PageAccess } from "@/lib/graphql/page-access";
import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { PageAccessReport, screenNeeds } from "../page-access-report";

afterEach(cleanup);

// Written from schema/page_access.graphqls: a screen whose route needs
// shipment read, that calls one endpoint needing shipment create and another
// needing shipment read again, and shows customers only through their view.
const access: PageAccess = {
  path: "/shipments",
  name: "Shipments",
  operations: ["ShipmentList", "ShipmentDetail"],
  requires: [
    {
      resource: "shipment",
      resourceName: "Shipment",
      operation: "read",
      operationName: "View",
      allowed: true,
    },
  ],
  endpoints: [
    {
      method: "POST",
      path: "/shipments/",
      requires: [
        {
          resource: "shipment",
          resourceName: "Shipment",
          operation: "create",
          operationName: "Create",
          allowed: false,
        },
      ],
    },
    {
      method: "GET",
      path: "/shipments/:param/",
      requires: [
        {
          resource: "shipment",
          resourceName: "Shipment",
          operation: "read",
          operationName: "View",
          allowed: true,
        },
      ],
    },
  ],
  kinds: [
    {
      resource: "shipment",
      resourceName: "Shipment",
      use: "REQUIRED",
      canRead: true,
      withheld: 0,
      fields: [],
    },
    {
      resource: "customer",
      resourceName: "Customer",
      use: "REFERENCE",
      canRead: false,
      withheld: 2,
      fields: [
        { name: "name", inView: true, decision: "VISIBLE" },
        { name: "creditLimit", inView: false, decision: "WITHHELD_KIND" },
        { name: "ShipmentMove.status", inView: false, decision: "WITHHELD_SENSITIVITY" },
      ],
    },
    {
      resource: "worker",
      resourceName: "Worker",
      use: "REFERENCE",
      canRead: false,
      withheld: 0,
      fields: [{ name: "wholeName", inView: true, decision: "VISIBLE" }],
    },
  ],
  withheld: 2,
  blocked: false,
};

describe("screenNeeds", () => {
  it("lists each permission once, the route's first, then the first endpoint asking", () => {
    expect(screenNeeds(access).map((need) => [need.resource, need.operation, need.source])).toEqual(
      [
        ["shipment", "read", ""],
        ["shipment", "create", "POST /shipments/"],
      ],
    );
  });
});

describe("PageAccessReport", () => {
  it("names each permission the screen needs and whether the reader holds it", () => {
    render(<PageAccessReport access={access} />);

    const needs = within(screen.getByRole("region", { name: "What this screen needs" }));
    const rows = needs.getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent("View on Shipment");
    expect(rows[0]).toHaveTextContent("Held");
    expect(rows[1]).toHaveTextContent("Create on Shipment");
    expect(rows[1]).toHaveTextContent("POST /shipments/");
    expect(rows[1]).toHaveTextContent("Missing");
    expect(screen.getByText("1 permission missing")).toBeInTheDocument();
  });

  it("lists the linked fields that come back as reference view only", () => {
    render(<PageAccessReport access={access} />);

    const records = within(screen.getByRole("region", { name: "Records on this screen" }));
    const customer = records.getByText("Customer").closest("li");
    expect(customer).not.toBeNull();
    const withheld = within(customer!).getByRole("list", { name: "Withheld fields" });
    expect(
      within(withheld)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual(["creditLimit Needs read", "ShipmentMove.status Above ceiling"]);
    expect(within(customer!).getByText("Reference view")).toBeInTheDocument();
    expect(within(customer!).getByText("No read")).toBeInTheDocument();

    const worker = records.getByText("Worker").closest("li");
    expect(worker).toHaveTextContent("1 linked field shown");
    expect(screen.getByText("2 linked fields withheld")).toBeInTheDocument();
  });

  it("warns when the screen would not open for the reader", () => {
    const { rerender } = render(<PageAccessReport access={access} />);
    expect(screen.queryByRole("alert")).toBeNull();
    expect(screen.queryByText(/would not open/)).toBeNull();

    rerender(<PageAccessReport access={{ ...access, blocked: true }} />);
    expect(screen.getByText(/This screen would not open, or would open empty/)).toBeInTheDocument();
  });

  it("says so when the screen checks nothing and shows no kind", () => {
    render(
      <PageAccessReport
        access={{ ...access, requires: [], endpoints: [], kinds: [], operations: [], withheld: 0 }}
      />,
    );

    expect(
      screen.getByText("Its route and endpoints check no permission of their own."),
    ).toBeInTheDocument();
    expect(
      screen.getByText("The screen shows no kind of record the permission registry knows."),
    ).toBeInTheDocument();
    expect(screen.queryByText(/GraphQL operation/)).toBeNull();
  });
});
