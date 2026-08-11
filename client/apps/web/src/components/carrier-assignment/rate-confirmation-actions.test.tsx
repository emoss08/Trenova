import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import type { RateConfirmation } from "@trenova/shared/types/rate-confirmation";
import { RateConfirmationActions } from "./rate-confirmation-actions";

const mocks = vi.hoisted(() => ({
  listByMove: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

vi.mock("@/services/api", () => ({
  apiService: {
    rateConfirmationService: {
      listByMove: mocks.listByMove,
      generate: vi.fn(),
      send: vi.fn(),
      confirm: vi.fn(),
      void: vi.fn(),
    },
  },
}));

function rateCon(overrides: Partial<RateConfirmation> = {}): RateConfirmation {
  return {
    id: "ratecon_01",
    organizationId: "org_01",
    businessUnitId: "bu_01",
    carrierAssignmentId: "casn_01",
    carrierId: "car_01",
    shipmentId: "shp_01",
    shipmentMoveId: "smv_01",
    revision: 1,
    status: "Confirmed",
    documentId: null,
    sentAt: null,
    sentToEmails: null,
    confirmedAt: 1_720_000_000,
    confirmedByName: "Jane Smith",
    voidedAt: null,
    voidReason: null,
    payloadSnapshot: null,
    generatedById: null,
    generatedVia: "Dispatcher",
    sourceTenderOfferId: null,
    confirmedVia: "Dispatcher",
    confirmedByTitle: null,
    version: 1,
    createdAt: 1_719_990_000,
    updatedAt: 1_720_000_000,
    ...overrides,
  };
}

function renderActions() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }
  return render(<RateConfirmationActions moveId="smv_01" carrierAssignmentId="casn_01" />, {
    wrapper: Wrapper,
  });
}

describe("RateConfirmationActions provenance badge", () => {
  beforeEach(() => {
    mocks.listByMove.mockReset();
  });

  it("shows how a tender-accepted rate confirmation was signed, with the signer's title", async () => {
    mocks.listByMove.mockResolvedValue([
      rateCon({
        generatedVia: "TenderAcceptance",
        sourceTenderOfferId: "tof_01",
        confirmedVia: "TenderAcceptance",
        confirmedByName: "Knight Logistics LLC",
        confirmedByTitle: null,
      }),
    ]);

    renderActions();

    expect(await screen.findByText("Signed via tender acceptance")).toBeInTheDocument();
    expect(screen.getByText("by Knight Logistics LLC")).toBeInTheDocument();
  });

  it("shows a publicly signed confirmation with the signer's title appended", async () => {
    mocks.listByMove.mockResolvedValue([
      rateCon({
        confirmedVia: "PublicSignature",
        confirmedByName: "Jane Smith",
        confirmedByTitle: "Operations Manager",
      }),
    ]);

    renderActions();

    expect(await screen.findByText("Signed publicly")).toBeInTheDocument();
    expect(screen.getByText("by Jane Smith, Operations Manager")).toBeInTheDocument();
  });

  it("shows dispatcher-recorded confirmations distinctly", async () => {
    mocks.listByMove.mockResolvedValue([rateCon()]);

    renderActions();

    expect(await screen.findByText("Confirmed by dispatcher")).toBeInTheDocument();
  });

  it("omits the provenance badge when the rate confirmation predates provenance tracking", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ confirmedVia: null })]);

    renderActions();

    expect(await screen.findByText("by Jane Smith")).toBeInTheDocument();
    expect(screen.queryByText("Confirmed by dispatcher")).not.toBeInTheDocument();
    expect(screen.queryByText("Signed publicly")).not.toBeInTheDocument();
  });
});
