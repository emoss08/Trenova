import { cleanup, render, screen } from "@testing-library/react";
import { PlanLocked } from "@trenova/shared/components/plan-locked";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, type PermissionManifest } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it } from "vitest";

// GET /me/permissions: the plan leaves Factoring (Accounting & Cash) out. Someone who may
// update the organization manages what it buys.
function holdManifest({ manages }: { manages: boolean }) {
  usePermissionStore.setState({
    manifest: {
      permissions: manages ? { organization: Operation.Read | Operation.Update } : {},
      routeAccess: {},
      unavailableFeatures: [
        {
          feature: "tms.factoring",
          featureName: "Factoring",
          addOn: "ACC",
          addOnName: "Accounting & Cash",
        },
      ],
    } as unknown as PermissionManifest,
  });
}

afterEach(() => {
  cleanup();
  usePermissionStore.getState().clearPermissions();
});

describe("PlanLocked", () => {
  it("renders nothing for a feature on the plan, or before the manifest loads", () => {
    const { container, rerender } = render(<PlanLocked feature="tms.factoring" variant="page" />);
    expect(container).toBeEmptyDOMElement();

    holdManifest({ manages: false });
    rerender(<PlanLocked feature="tms.billing" variant="page" />);
    expect(container).toBeEmptyDOMElement();
  });

  it("draws a card with the item's name, the add-on and one line", () => {
    holdManifest({ manages: false });
    const { container } = render(
      <PlanLocked feature="tms.factoring" variant="card" title="Factored invoices" />,
    );

    const card = screen.getByRole("region", { name: "Factored invoices" });
    expect(card).toHaveAttribute("data-variant", "card");
    expect(screen.getByText("Part of Accounting & Cash")).toBeInTheDocument();
    expect(screen.getByText("Not on your plan")).toBeInTheDocument();
    expect(container.querySelector("svg")).not.toBeNull();
  });

  it("says one sentence inline", () => {
    holdManifest({ manages: false });
    const { container } = render(<PlanLocked feature="tms.factoring" variant="inline" />);

    expect(container.textContent).toBe(
      "Factoring is part of the Accounting & Cash add-on, which is not on your plan.",
    );
  });

  it("explains a page and sends the person to an administrator", () => {
    holdManifest({ manages: false });
    render(<PlanLocked feature="tms.factoring" variant="page" title="Factoring companies" />);

    expect(screen.getByRole("heading", { name: "Not on your plan" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "Factoring companies is part of the Accounting & Cash add-on, which is not on your plan.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Ask your organization's administrator about adding it."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("tells someone who manages the organization to add it, without a link", () => {
    holdManifest({ manages: true });
    render(<PlanLocked feature="tms.factoring" variant="page" />);

    expect(screen.getByText("Add it from your Trenova account.")).toBeInTheDocument();
    expect(screen.queryByText(/administrator/)).toBeNull();
    expect(screen.queryByRole("link")).toBeNull();
  });

  it.each(["card", "inline", "page"] as const)(
    "never shows the feature key or the add-on code (%s)",
    (variant) => {
      holdManifest({ manages: false });
      const { container } = render(
        <PlanLocked feature="tms.factoring" variant={variant} title="Factoring" />,
      );

      expect(container.textContent).not.toContain("tms.factoring");
      expect(container.textContent).not.toMatch(/\bACC\b/);
    },
  );
});
