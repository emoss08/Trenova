import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { buildCatalogIndex } from "../builder-state";
import { CatalogFieldTree } from "../catalog-field-tree";
import { catalog } from "./catalog-access-fixture";

function renderTree() {
  const onSelectField = vi.fn();
  render(
    <CatalogFieldTree
      index={buildCatalogIndex(catalog)}
      entityKey="shipment"
      onSelectField={onSelectField}
    />,
  );
  return onSelectField;
}

describe("CatalogFieldTree", () => {
  it("offers a field the caller may use", async () => {
    const onSelectField = renderTree();

    await userEvent.click(screen.getByRole("button", { name: /PRO Number/ }));

    expect(onSelectField).toHaveBeenCalledWith(
      expect.objectContaining({ ref: { path: undefined, field: "proNumber" } }),
    );
  });

  it("shows a protected field with its group and disables it without the group", () => {
    renderTree();

    const margin = screen.getByRole("button", { name: /Margin Floor/ });
    expect(margin).toBeDisabled();
    expect(margin).toHaveTextContent("Margins");
    expect(margin).toHaveTextContent("Needs the Margins field group");

    const license = screen.getByRole("button", { name: /Driver License/ });
    expect(license).toBeEnabled();
    expect(license).toHaveTextContent("Personal");
  });

  it("shows a field above the caller's ceiling as unavailable", () => {
    renderTree();

    const status = screen.getByRole("button", { name: /Status/ });
    expect(status).toBeDisabled();
    expect(status).toHaveTextContent("Unavailable at your role's sensitivity level");
  });

  it("joins a kind the caller cannot read through its reference view alone", async () => {
    const onSelectField = renderTree();

    const edge = screen.getByRole("button", { name: /Customer/ });
    expect(edge).toHaveTextContent("Reference only");
    await userEvent.click(edge);

    expect(screen.getByText("Reference only — the rest needs Customers read")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /DOT Number/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Code/ })).toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: /^Name/ }));
    expect(onSelectField).toHaveBeenCalledWith(
      expect.objectContaining({ ref: { path: ["customer"], field: "name" } }),
    );
  });

  it("lists an edge the caller cannot follow, locked, with the read it needs", () => {
    renderTree();

    const locked = screen.getByTitle("Needs Workers read");
    expect(within(locked).getByText("Driver Profile")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Driver Profile/ })).not.toBeInTheDocument();
  });
});
