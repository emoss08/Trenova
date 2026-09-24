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
    core: false,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
    ...overrides,
  };
}

const catalog = [
  tool({ name: "get_shipment" }),
  tool({ name: "cancel_shipment", kind: "action" }),
  tool({ name: "list_workers", resource: "worker" }),
  tool({ name: "recall_memory", resource: "agent_memory", core: true }),
  tool({ name: "list_reports", resource: "report" }),
  tool({
    name: "create_dashboard",
    kind: "action",
    resource: "report_dashboard",
    prerequisites: ["list_reports"],
  }),
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

  // An agent with no task tools still recalls and escalates, so "answers
  // only" would be false.
  it("says what an agent with no task tools can still do", () => {
    renderSummary([]);

    expect(screen.getByText(/no task tools/i)).toBeInTheDocument();
    expect(screen.queryByText(/answers only/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /choose tools/i })).toBeInTheDocument();
  });

  it("shows the always-on tools, and none can be removed", () => {
    renderSummary(["get_shipment"]);

    const alwaysOn = screen.getByRole("list", { name: /always on/i });
    expect(alwaysOn).toHaveTextContent(/recall/i);
    expect(screen.queryByRole("button", { name: /remove recall/i })).not.toBeInTheDocument();
  });

  it("says which reads come with a chosen tool", () => {
    renderSummary(["create_dashboard"]);

    const included = screen.getByRole("list", { name: /included with your tools/i });
    expect(included).toHaveTextContent(/browse reports/i);
  });

  it("never offers a core tool as a choice, even through All reads", async () => {
    const { onSelectedChange } = renderSummary([]);

    await userEvent.click(screen.getByRole("button", { name: /choose tools/i }));
    await screen.findByRole("dialog");
    expect(screen.queryByRole("checkbox", { name: /recall/i })).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /all reads/i }));
    expect(onSelectedChange).toHaveBeenCalledWith(["get_shipment", "list_workers", "list_reports"]);
  });
});
