import { stubLayout } from "@/test/layout";
import type { ToolCatalogEntry } from "@/types/assistant";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToolSummary } from "../tool-summary";

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function tool(overrides: Partial<ToolCatalogEntry>): ToolCatalogEntry {
  return {
    name: "get_shipment",
    description: "Reads one shipment",
    parameters: null,
    kind: "query",
    resource: "shipment",
    operation: "read",
    defaultAutonomyTier: "",
    reversible: false,
    ...overrides,
  };
}

const catalog = [
  tool({ name: "get_shipment" }),
  tool({ name: "cancel_shipment", kind: "action" }),
  tool({ name: "list_workers", resource: "worker" }),
];

function renderSummary(selected: string[]) {
  const onSelectedChange = vi.fn();
  const onTiersChange = vi.fn();
  render(
    <ToolSummary
      tools={catalog}
      selected={selected}
      tiers={{}}
      ceiling="ActWithApproval"
      onSelectedChange={onSelectedChange}
      onTiersChange={onTiersChange}
    />,
  );

  return { onSelectedChange, onTiersChange };
}

/**
 * The catalog used to sit open in the form: fifty rows in twenty-six groups.
 * The form now shows only what was chosen, and the catalog opens on request.
 */
describe("ToolSummary", () => {
  it("keeps the catalog out of the form until it is asked for", async () => {
    renderSummary(["get_shipment"]);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.queryByText("Lists drivers")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /change tools/i }));

    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: /tool groups/i })).toBeInTheDocument();
  });

  it("sums up what was chosen and lets a tool be dropped in place", async () => {
    const { onSelectedChange } = renderSummary(["get_shipment", "cancel_shipment"]);

    expect(screen.getByText("1 read · 1 change")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /remove cancel shipment/i }));

    expect(onSelectedChange).toHaveBeenCalledWith(["get_shipment"]);
  });

  it("says plainly when the agent can only answer", () => {
    renderSummary([]);

    expect(screen.getByText(/answers only/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /choose tools/i })).toBeInTheDocument();
  });
});
