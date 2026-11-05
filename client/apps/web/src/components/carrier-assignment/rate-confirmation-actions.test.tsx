import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { RateConfirmation } from "@trenova/shared/types/rate-confirmation";
import { RateConfirmationActions } from "./rate-confirmation-actions";

const mocks = vi.hoisted(() => ({
  listByMove: vi.fn(),
  send: vi.fn(),
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

vi.mock("@/services/rate-confirmation", () => ({
  rateConfirmationService: {
    listByMove: mocks.listByMove,
    generate: vi.fn(),
    send: mocks.send,
    confirm: vi.fn(),
    void: vi.fn(),
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

  it("marks a rate confirmation that the tender acceptance issued on its own", async () => {
    mocks.listByMove.mockResolvedValue([
      rateCon({
        generatedVia: "TenderAcceptance",
        sourceTenderOfferId: "tof_01",
      }),
    ]);

    renderActions();

    const badge = await screen.findByText("Auto-issued from tender");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute(
      "title",
      "Generated automatically when the carrier accepted the tender",
    );
  });

  it("omits the auto-issue badge for a dispatcher-generated rate confirmation", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ generatedVia: "Dispatcher" })]);

    renderActions();

    expect(await screen.findByText("Confirmed by dispatcher")).toBeInTheDocument();
    expect(screen.queryByText("Auto-issued from tender")).not.toBeInTheDocument();
  });

  it("omits the provenance badge when the rate confirmation predates provenance tracking", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ confirmedVia: null })]);

    renderActions();

    expect(await screen.findByText("by Jane Smith")).toBeInTheDocument();
    expect(screen.queryByText("Confirmed by dispatcher")).not.toBeInTheDocument();
    expect(screen.queryByText("Signed publicly")).not.toBeInTheDocument();
  });
});

async function menuItems(): Promise<string[]> {
  await userEvent.click(await screen.findByRole("button", { name: "Rate confirmation actions" }));
  const items = await screen.findAllByRole("menuitem");
  return items.map((item) => item.textContent ?? "");
}

describe("RateConfirmationActions next step", () => {
  beforeEach(() => {
    mocks.listByMove.mockReset();
  });

  it("offers to generate one when none exists, with nothing else to do", async () => {
    mocks.listByMove.mockResolvedValue([]);

    renderActions();

    expect(await screen.findByText("Not generated yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Generate" })).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Rate confirmation actions" }),
    ).not.toBeInTheDocument();
  });

  it("offers to send a generated one from the menu alone", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ status: "Generated", confirmedVia: null })]);

    renderActions();

    expect(await screen.findByText("Revision 1")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send" })).not.toBeInTheDocument();
    const items = await menuItems();
    expect(items.some((item) => item.startsWith("Send"))).toBe(true);
    expect(items.some((item) => item.startsWith("Mark confirmed"))).toBe(true);
    expect(items.some((item) => item.startsWith("Void"))).toBe(true);
  });

  it("offers to mark a sent one confirmed, with resending in the menu", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ status: "Sent", confirmedVia: null })]);

    renderActions();

    expect(await screen.findByRole("button", { name: "Mark confirmed" })).toBeInTheDocument();
    const items = await menuItems();
    expect(items.some((item) => item.startsWith("Mark confirmed"))).toBe(false);
    expect(items.some((item) => item.startsWith("Resend"))).toBe(true);
  });

  it("asks nothing more of a confirmed one, and offers its PDF from the menu", async () => {
    mocks.listByMove.mockResolvedValue([rateCon({ documentId: "doc_01" })]);

    renderActions();

    expect(await screen.findByText("Revision 1")).toBeInTheDocument();
    for (const name of ["Generate", "Send", "Mark confirmed"]) {
      expect(screen.queryByRole("button", { name })).not.toBeInTheDocument();
    }
    const items = await menuItems();
    expect(items.some((item) => item.startsWith("View PDF"))).toBe(true);
    expect(items.some((item) => item.startsWith("Regenerate"))).toBe(true);
    expect(items.some((item) => /^Send|^Resend|^Mark confirmed/.test(item))).toBe(false);
  });
});

describe("RateConfirmationActions while a send is out", () => {
  beforeEach(() => {
    mocks.listByMove.mockReset();
    mocks.send.mockReset();
  });

  it("sends once, however often Send is chosen, and holds the menu until it lands", async () => {
    let land!: (value: RateConfirmation) => void;
    mocks.send.mockImplementation(
      () =>
        new Promise<RateConfirmation>((resolve) => {
          land = resolve;
        }),
    );
    mocks.listByMove.mockResolvedValue([rateCon({ status: "Generated", confirmedVia: null })]);

    renderActions();

    await userEvent.click(await screen.findByRole("button", { name: "Rate confirmation actions" }));
    const send = await screen.findByRole("menuitem", { name: /^Send/ });
    await userEvent.click(send);

    const trigger = screen.getByRole("button", { name: "Rate confirmation actions" });
    await waitFor(() => expect(trigger).toBeDisabled());
    expect(trigger).toHaveAttribute("aria-busy", "true");
    await userEvent.click(trigger);
    expect(screen.queryByRole("menuitem", { name: /^Send/ })).not.toBeInTheDocument();
    expect(mocks.send).toHaveBeenCalledTimes(1);

    await act(async () => {
      land(rateCon({ status: "Sent", confirmedVia: null }));
    });

    await waitFor(() => expect(trigger).toBeEnabled());
    expect(mocks.send).toHaveBeenCalledTimes(1);
  });
});
