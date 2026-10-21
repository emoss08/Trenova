import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { LinkedRecordLink } from "../linked-record-link";

afterEach(() => {
  cleanup();
  clearPermissions();
});

function renderLink() {
  return render(
    <MemoryRouter>
      <LinkedRecordLink kind="customer" id="cus_01" extra={{ tab: "tracking" }} className="lnk">
        Acme Foods
      </LinkedRecordLink>
    </MemoryRouter>,
  );
}

describe("LinkedRecordLink", () => {
  it("links to the record's own page for someone who may read its kind", () => {
    seedPermissions({ customer: Operation.Read });
    renderLink();

    const link = screen.getByRole("link", { name: "Acme Foods" });
    expect(link.getAttribute("href")).toBe(
      "/billing/configuration-files/customers?panelType=edit&panelEntityId=cus_01&tab=tracking",
    );
    expect(link.className).toBe("lnk");
  });

  it("draws the name as plain text with a lock for someone without the kind's read", () => {
    seedPermissions({ shipment: Operation.Read, customer: Operation.Update });
    const { container } = renderLink();

    expect(screen.queryByRole("link")).toBeNull();
    const reference = container.querySelector('[data-slot="locked-reference"]');
    expect(reference?.textContent).toBe("Acme Foods");
    expect(reference?.getAttribute("aria-description")).toBe("You don't have access to customers");
  });

  it("draws no link before the person's permissions are known", () => {
    renderLink();

    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText("Acme Foods")).toBeTruthy();
  });
});
