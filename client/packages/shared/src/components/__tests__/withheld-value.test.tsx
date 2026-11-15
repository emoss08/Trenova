import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LockedReference, WithheldValue } from "@trenova/shared/components/withheld-value";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { PermissionManifest } from "@trenova/shared/types/permission";

afterEach(cleanup);

describe("WithheldValue", () => {
  it("says the value is hidden and whose read it needs, never a dash or a zero", () => {
    const { container } = render(<WithheldValue kind="customer" />);

    const value = screen.getByText("Hidden").closest('[data-slot="locked-reference"]');
    expect(value).not.toBeNull();
    expect(value?.getAttribute("aria-description")).toBe("You don't have access to customers");
    expect(value?.getAttribute("tabindex")).toBe("0");
    expect(container.textContent).toBe("Hidden");
    expect(container.querySelector('[data-slot="locked-reference-icon"]')).not.toBeNull();
  });

  it("names the protected field group a withheld margin needs", () => {
    const { container } = render(<WithheldValue group="Margins" />);

    expect(container.textContent).toBe("Hidden");
    expect(
      screen
        .getByText("Hidden")
        .closest('[data-slot="locked-reference"]')
        ?.getAttribute("aria-description"),
    ).toBe("You don't have access to margins");
  });

  it("names a kind without a label of its own in words", () => {
    render(<WithheldValue kind="agent_memory" />);

    expect(
      screen
        .getByText("Hidden")
        .closest('[data-slot="locked-reference"]')
        ?.getAttribute("aria-description"),
    ).toBe("You don't have access to agent memory");
  });
});

describe("LockedReference", () => {
  it("draws a linked record's name as plain text after the lock", () => {
    const { container } = render(<LockedReference kind="worker">Rosa Diaz</LockedReference>);

    const reference = container.querySelector('[data-slot="locked-reference"]');
    expect(reference?.tagName).toBe("SPAN");
    expect(reference?.querySelector("a")).toBeNull();
    expect(reference?.firstElementChild?.getAttribute("data-slot")).toBe("locked-reference-icon");
    expect(reference?.textContent).toBe("Rosa Diaz");
    expect(reference?.getAttribute("aria-description")).toBe("You don't have access to workers");
  });

  it("keeps a row's own layout and ends it with the lock", () => {
    const { container } = render(
      <LockedReference kind="tractor" layout="row" className="dk-rec-row">
        <b>T-101</b>
        <span>Tractor</span>
      </LockedReference>,
    );

    const row = container.querySelector('[data-slot="locked-reference"]');
    expect(row?.tagName).toBe("DIV");
    expect(row?.className).toContain("dk-rec-row");
    expect(row?.firstElementChild?.tagName).toBe("B");
    expect(row?.lastElementChild?.getAttribute("data-slot")).toBe("locked-reference-icon");
  });
});

describe("WithheldValue for a value the plan withheld", () => {
  afterEach(() => usePermissionStore.getState().clearPermissions());

  function holdManifest() {
    usePermissionStore.setState({
      manifest: {
        permissions: {},
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

  it("names the add-on in the same compact form as Hidden, from a feature key", () => {
    holdManifest();
    const { container } = render(<WithheldValue plan="tms.factoring" />);

    const value = container.querySelector('[data-slot="locked-reference"]');
    expect(value?.getAttribute("data-withheld")).toBe("plan");
    expect(container.textContent).toBe("Part of Accounting & Cash");
    expect(value?.getAttribute("aria-description")).toBe(
      "Factoring is part of the Accounting & Cash add-on, which is not on your plan.",
    );
    expect(container.querySelector('[data-slot="locked-reference-icon"]')).not.toBeNull();
  });

  it("takes a response's withholding for its path", () => {
    holdManifest();
    render(<WithheldValue plan={{ path: "invoice.factoringCompany", feature: "tms.factoring" }} />);

    expect(screen.getByText("Part of Accounting & Cash")).toBeInTheDocument();
  });

  it("says only that it is not on the plan when the manifest does not list the feature", () => {
    const { container } = render(<WithheldValue plan="tms.factoring" />);

    expect(container.textContent).toBe("Not on your plan");
    expect(container.textContent).not.toContain("tms.factoring");
  });
});
