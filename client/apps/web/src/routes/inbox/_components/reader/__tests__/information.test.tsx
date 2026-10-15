import type { FactoringRemittance } from "@/lib/graphql/factoring-remittance";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Information } from "../information";

const mocks = vi.hoisted(() => ({
  listFactoringRemittances: vi.fn(),
  canRead: true,
}));

vi.mock("@/lib/graphql/factoring-remittance", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-remittance")>()),
  listFactoringRemittances: mocks.listFactoringRemittances,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canRead, isLoading: false }),
}));
vi.mock("../../use-triage-actions", () => ({ useTriageActions: () => ({}) }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.canRead = true;
});

function message(overrides: Partial<InboundMessageDetail> = {}): InboundMessageDetail {
  return {
    id: "inmsg_1",
    classification: "FactorRemittance",
    matchedShipment: null,
    matchedCustomer: null,
    facts: [],
    attachments: [],
    ...overrides,
  } as InboundMessageDetail;
}

function remittance(overrides: Partial<FactoringRemittance>): FactoringRemittance {
  return {
    id: "facrem_1",
    factoringCompanyId: "fac_rts",
    fileName: "rts-funding.csv",
    status: "Parsed",
    factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
    ...overrides,
  } as FactoringRemittance;
}

function Location() {
  return <p data-testid="location">{useLocation().pathname}</p>;
}

function renderInformation(detail: InboundMessageDetail) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/inbox"]}>
        <Routes>
          <Route path="/inbox" element={<Information message={detail} />} />
          <Route path="*" element={<Location />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Information remittance rows", () => {
  it("lists every remittance the email was read into and opens the one chosen", async () => {
    mocks.listFactoringRemittances.mockResolvedValue({
      remittances: [
        remittance({}),
        remittance({
          id: "facrem_2",
          fileName: "rts-reserve.pdf",
          status: "NeedsMapping",
        }),
      ],
      totalCount: 2,
    });

    renderInformation(message());

    expect(await screen.findByText("rts-funding.csv")).toBeTruthy();
    expect(screen.getByText("rts-reserve.pdf")).toBeTruthy();
    expect(screen.getAllByText("Remittance · RTS Financial")).toHaveLength(2);
    expect(mocks.listFactoringRemittances).toHaveBeenCalledWith(
      { inboundMessageId: "inmsg_1" },
      expect.anything(),
    );

    await userEvent.click(screen.getByText("rts-reserve.pdf"));
    expect(screen.getByTestId("location").textContent).toBe(
      "/billing/factoring/remittances/facrem_2",
    );
  });

  it("reads no remittances for someone who cannot see them", () => {
    mocks.canRead = false;

    const { container } = renderInformation(message());

    expect(container.textContent).toBe("");
    expect(mocks.listFactoringRemittances).not.toHaveBeenCalled();
  });

  it("reads no remittances for a message of another kind", () => {
    const { container } = renderInformation(message({ classification: "RateConfirmation" }));

    expect(container.textContent).toBe("");
    expect(mocks.listFactoringRemittances).not.toHaveBeenCalled();
  });
});
