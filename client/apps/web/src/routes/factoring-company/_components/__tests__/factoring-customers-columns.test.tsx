import type { FactoringNoa } from "@/lib/graphql/factoring";
import {
  createFactoringCompanyCustomersTableConfig,
  type FactoringCompanyCustomer,
} from "@/lib/graphql/factoring-company-page";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getFactoringCustomerColumns,
  type FactoringCustomerActions,
} from "../factoring-customers-columns";

afterEach(cleanup);

const NOW = 1_762_560_000;
const t = ((message: string) => message) as never;

const CUSTOMER: FactoringCompanyCustomer = {
  id: "cus_1",
  code: "ACME",
  name: "Acme Foods",
  status: "Active",
  notice: null,
  lastReleasedAt: null,
  fundedCount: 3,
  fundedMinor: 1_250_000,
} as FactoringCompanyCustomer;

function noa(status: FactoringNoa["status"], extra: Partial<FactoringNoa> = {}): FactoringNoa {
  return {
    id: "noa_1",
    status,
    createdAt: NOW - 86_400,
    sentAt: status === "Draft" ? null : NOW - 86_400,
    acknowledgedAt: status === "Acknowledged" ? NOW - 3_600 : null,
    expiresAt: null,
    sentTo: "ap@acme.example",
    ...extra,
  } as FactoringNoa;
}

function actions(): FactoringCustomerActions {
  return {
    canCreate: true,
    canUpdate: true,
    onDraft: vi.fn(),
    onSend: vi.fn(),
    onAcknowledge: vi.fn(),
    onRelease: vi.fn(),
  };
}

function cell(
  id: string,
  customer: FactoringCompanyCustomer,
  handlers: FactoringCustomerActions,
  canReadMoney = true,
) {
  const column = getFactoringCustomerColumns(t, {
    nowUnix: NOW,
    canReadMoney,
    actions: handlers,
  }).find(
    (candidate) =>
      candidate.id === id || ("accessorKey" in candidate && candidate.accessorKey === id),
  );
  const render = column?.cell as (context: {
    row: { original: FactoringCompanyCustomer };
  }) => ReactNode;
  return render({ row: { original: customer } });
}

function renderCell(node: ReactNode) {
  return render(<MemoryRouter>{node}</MemoryRouter>);
}

describe("factoring customer columns", () => {
  it("offers a draft for a customer with no notice and hands over the customer", () => {
    const handlers = actions();
    renderCell(cell("actions", CUSTOMER, handlers));

    fireEvent.click(screen.getByRole("button", { name: "Draft notice" }));
    expect(handlers.onDraft).toHaveBeenCalledWith("cus_1");
  });

  it("moves a drafted notice on by sending it", () => {
    const handlers = actions();
    const draft = noa("Draft");
    renderCell(cell("actions", { ...CUSTOMER, notice: draft }, handlers));

    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    expect(handlers.onSend).toHaveBeenCalledWith(draft);
    expect(screen.queryByRole("button", { name: "Draft notice" })).toBeNull();
  });

  it("records the acknowledgement of a sent notice, or sends it again", () => {
    const handlers = actions();
    const sent = noa("Sent");
    renderCell(cell("actions", { ...CUSTOMER, notice: sent }, handlers));

    fireEvent.click(screen.getByRole("button", { name: "Record acknowledgement" }));
    expect(handlers.onAcknowledge).toHaveBeenCalledWith(sent);
    fireEvent.click(screen.getByRole("button", { name: "Send again" }));
    expect(handlers.onSend).toHaveBeenCalledWith(sent);
  });

  it("releases an acknowledged notice", () => {
    const handlers = actions();
    const acknowledged = noa("Acknowledged");
    renderCell(cell("actions", { ...CUSTOMER, notice: acknowledged }, handlers));

    fireEvent.click(screen.getByRole("button", { name: "Release" }));
    expect(handlers.onRelease).toHaveBeenCalledWith(acknowledged);
  });

  it("offers nothing to someone who may not change notices", () => {
    renderCell(cell("actions", CUSTOMER, { ...actions(), canCreate: false, canUpdate: false }));

    expect(screen.queryByRole("button")).toBeNull();
  });

  it("withholds the funded amount without reserve access but keeps the invoice count", () => {
    renderCell(cell("funded", CUSTOMER, actions(), false));

    expect(screen.getByText("—")).toBeTruthy();
    expect(screen.queryByText("$12,500.00")).toBeNull();
  });
});

describe("createFactoringCompanyCustomersTableConfig", () => {
  it("scopes the table to the factor and the notice stage", () => {
    expect(
      createFactoringCompanyCustomersTableConfig("fac_1", "NeedsNotice").extraVariables,
    ).toEqual({ factoringCompanyId: "fac_1", stage: "NeedsNotice" });
    expect(createFactoringCompanyCustomersTableConfig("fac_1", null).extraVariables).toEqual({
      factoringCompanyId: "fac_1",
      stage: null,
    });
  });
});
