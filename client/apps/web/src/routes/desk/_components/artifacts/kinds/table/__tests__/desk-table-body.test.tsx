import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskTableBody } from "../desk-table-body";

const card = vi.hoisted(() => ({ opened: [] as string[] }));

vi.mock("../../../cards/formula-template/desk-formula-template", () => ({
  DeskFormulaTemplate: ({
    recordId,
    order,
    onSelect,
    onBack,
  }: {
    recordId: string;
    order: readonly string[];
    onSelect: (id: string) => void;
    onBack: () => void;
  }) => {
    card.opened.push(recordId);
    const next = order[order.indexOf(recordId) + 1];
    return (
      <div data-testid="formula-card">
        <span data-testid="card-id">{recordId}</span>
        <span data-testid="card-order">{order.join(",")}</span>
        <button type="button" onClick={onBack}>
          Back
        </button>
        <button type="button" onClick={() => next && onSelect(next)}>
          Next
        </button>
      </div>
    );
  },
}));

vi.mock("../../../cards/shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    order,
    onBack,
  }: {
    recordId: string;
    order: readonly string[];
    onBack: () => void;
  }) => {
    card.opened.push(recordId);
    return (
      <div data-testid="shipment-card">
        <span data-testid="shipment-id">{recordId}</span>
        <span data-testid="shipment-order">{order.join(",")}</span>
        <button type="button" onClick={onBack}>
          Back to the shipments
        </button>
      </div>
    );
  },
  DeskShipmentArtifact: () => null,
}));

vi.mock("../../../cards/billing-queue-item/desk-billing-record", () => ({
  DeskBillingRecord: ({ recordId, onBack }: { recordId: string; onBack: () => void }) => (
    <div data-testid="billing-card">
      <span>Item {recordId}</span>
      <button type="button" onClick={onBack}>
        Back to the queue
      </button>
    </div>
  ),
  DeskBillingArtifact: () => null,
}));

vi.mock("../../../cards/billing-queue-item/desk-billing-table", async (importOriginal) => ({
  ...(await importOriginal<
    typeof import("../../../cards/billing-queue-item/desk-billing-table")
  >()),
  useBillingSummaries: () => new Map(),
}));

function artifact(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "table_view",
    status: "Ready",
    title: "Formula templates",
    payload,
    sourceToolCallId: "",
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug: "art_1",
    turn: "",
    createdAt: 1,
    updatedAt: 1,
  };
}

/** A formula template list as the server projects it: typed columns, and each row's id. */
function templates(): AssistantArtifact {
  return artifact({
    display: 1,
    tool: "formula_list_templates",
    entity: "formula_templates",
    recordEntity: "formula_template",
    columns: [
      { key: "name", label: "Name", type: "text" },
      { key: "type", label: "Type", type: "enum" },
      { key: "status", label: "Status", type: "status" },
    ],
    rows: [
      { id: "ft_1", name: "Reefer per mile", type: "FreightCharge", status: "Active" },
      { id: "ft_2", name: "Detention", type: "AccessorialCharge", status: "Draft" },
      { id: "ft_3", name: "Dry van per mile", type: "FreightCharge", status: "Active" },
    ],
    rowCount: 3,
  });
}

function Where() {
  const location = useLocation();
  return <div data-testid="where">{`${location.pathname}${location.search}`}</div>;
}

function renderTable(shown: AssistantArtifact) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/desk"]}>
        <Routes>
          <Route path="/desk" element={<DeskTableBody artifact={shown} previous={null} />} />
          <Route path="*" element={<Where />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  card.opened = [];
});

describe("DeskTableBody with formula templates", () => {
  it("opens a row as the template itself, with the rows in the order shown", async () => {
    renderTable(templates());

    fireEvent.click(screen.getByText("Detention"));

    expect(await screen.findByTestId("card-id")).toHaveTextContent("ft_2");
    expect(screen.getByTestId("card-order")).toHaveTextContent("ft_1,ft_2,ft_3");
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("steps through only the rows the filter left", async () => {
    renderTable(templates());

    fireEvent.change(screen.getByLabelText("Filter rows"), { target: { value: "per mile" } });
    fireEvent.click(screen.getByText("Dry van per mile"));

    expect(await screen.findByTestId("card-id")).toHaveTextContent("ft_3");
    expect(screen.getByTestId("card-order")).toHaveTextContent("ft_1,ft_3");
  });

  it("goes back to the table as it was left", async () => {
    renderTable(templates());

    fireEvent.change(screen.getByLabelText("Filter rows"), { target: { value: "per mile" } });
    fireEvent.click(screen.getByText("Reefer per mile"));
    fireEvent.click(await screen.findByRole("button", { name: "Back" }));

    expect(screen.queryByTestId("formula-card")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Filter rows")).toHaveValue("per mile");
    expect(screen.queryByText("Detention")).not.toBeInTheDocument();
    expect(screen.getByText("Dry van per mile")).toBeInTheDocument();
  });

  it("shows another template when the card picks one", async () => {
    renderTable(templates());

    fireEvent.click(screen.getByText("Reefer per mile"));
    fireEvent.click(await screen.findByRole("button", { name: "Next" }));

    expect(await screen.findByTestId("card-id")).toHaveTextContent("ft_2");
  });

  it("opens the Formula Studio from a row's link without opening the card", () => {
    renderTable(templates());

    const [first] = screen.getAllByRole("link", { name: "Open this record" });
    expect(first).toHaveAttribute(
      "href",
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
    fireEvent.click(first);

    expect(card.opened).toEqual([]);
    expect(screen.getByTestId("where")).toHaveTextContent(
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
  });

  it("still opens any other record on its page", () => {
    renderTable(
      artifact({
        display: 1,
        tool: "masterdata_list_commodities",
        entity: "commodities",
        recordEntity: "commodity",
        columns: [{ key: "name", label: "Name", type: "text" }],
        rows: [{ id: "com_1", name: "Avery Lane" }],
        rowCount: 1,
      }),
    );

    fireEvent.click(screen.getByText("Avery Lane"));

    expect(card.opened).toEqual([]);
    expect(screen.getByTestId("where")).toHaveTextContent(
      "/shipment-management/configuration-files/commodities",
    );
  });
});

describe("DeskTableBody with shipments", () => {
  it("opens a shipment row in place as the shipment, and goes back to the table", async () => {
    renderTable(
      artifact({
        display: 1,
        tool: "shipment_list",
        entity: "shipments",
        recordEntity: "shipment",
        columns: [{ key: "proNumber", label: "Pro number", type: "text" }],
        rows: [
          { id: "shp_1", proNumber: "PRO-1" },
          { id: "shp_2", proNumber: "PRO-2" },
        ],
        rowCount: 2,
      }),
    );

    fireEvent.click(screen.getByText("PRO-2"));

    expect(await screen.findByTestId("shipment-id")).toHaveTextContent("shp_2");
    expect(screen.getByTestId("shipment-order")).toHaveTextContent("shp_1,shp_2");
    expect(screen.queryByRole("table")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to the shipments" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
  });
});

describe("DeskTableBody with billing queue items", () => {
  it("opens a queue row in place as the billing item, and goes back to the queue", async () => {
    renderTable(
      artifact({
        display: 1,
        tool: "billing_list_queue_items",
        entity: "billing_queue_items",
        recordEntity: "billing_queue_item",
        columns: [
          { key: "number", label: "Number", type: "text" },
          { key: "status", label: "Status", type: "status" },
        ],
        rows: [
          { id: "bqi_1", number: "INV2610000079", status: "ReadyForReview" },
          { id: "bqi_2", number: "INV2610000044", status: "Approved" },
        ],
        rowCount: 2,
      }),
    );

    fireEvent.click(screen.getByText("INV2610000044"));
    expect(await screen.findByText("Item bqi_2")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to the queue" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(card.opened).toEqual([]);
  });
});
