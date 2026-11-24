import { TestRouter } from "@trenova/shared/testing/test-router";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { minorMoney } from "../../../record-card/record-card-values";
import { DeskBankReceipt } from "../desk-bank-receipt";
import { candidate, exceptionCard, matchedCard } from "./fixtures";

const service = vi.hoisted(() => ({ getCard: vi.fn() }));

vi.mock("@/services/bank-receipt", () => ({ bankReceiptService: service }));

// The invoice and customer cards are their own chunks; here each stands in as
// what it was given, with its way back.
function standIn(kind: string) {
  return ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>
        {kind} card {recordId}
      </p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  );
}
vi.mock("../../invoice/desk-invoice", () => ({
  DeskInvoice: standIn("Invoice"),
  DeskInvoiceArtifact: () => null,
}));
vi.mock("../../customer/desk-customer", () => ({
  DeskCustomer: standIn("Customer"),
  DeskCustomerArtifact: () => null,
}));

function draw(recordId = "brcpt_1") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <DeskBankReceipt recordId={recordId} onBack={vi.fn()} />
      </TestRouter>
    </QueryClientProvider>,
  );
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  seedPermissions({
    [Resource.BankReceipt]: Operation.Read,
    [Resource.CustomerPayment]: Operation.Read,
    [Resource.BankReceiptWorkItem]: Operation.Read,
    [Resource.Invoice]: Operation.Read,
    [Resource.Customer]: Operation.Read,
  });
  service.getCard
    .mockReset()
    .mockImplementation(async (id: string) => (id === "brcpt_2" ? matchedCard() : exceptionCard()));
});

afterEach(() => {
  clearPermissions();
});

describe("DeskBankReceipt", () => {
  it("holds the card's outline while the receipt is read, then draws what came in and why it is not applied", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the bank receipt…");
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "ACH 7731090", level: 2 }),
    ).toBeInTheDocument();
    expect(service.getCard).toHaveBeenCalledWith("brcpt_1", expect.objectContaining({}));
    expect(screen.getByText(minorMoney(945_000), { selector: "b" })).toBeInTheDocument();
    expect(screen.getByText("Exception")).toBeInTheDocument();
    const why = section("Whether it is applied");
    expect(within(why).getByText("No posted payment matches it")).toBeInTheDocument();
    expect(
      within(why).getByText("No unique customer payment match found for bank receipt"),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "The payment it was matched to" })).toBeNull();
    expect(within(section("Reconciliation queue")).getByText("Assigned")).toBeInTheDocument();
    expect(screen.queryByText(/to the cent/u)).toBeNull();
    expect(screen.getByRole("link", { name: "Open bank receipt" })).toHaveAttribute(
      "href",
      "/accounting/reconciliation/bank-receipts?item=brcpt_1",
    );
  });

  it("draws a matched receipt's payment and the invoices it went to, and opens them in the card's place", async () => {
    draw("brcpt_2");

    const paid = await screen.findByRole("heading", { name: "The payment it was matched to" });
    const body = paid.closest("section") as HTMLElement;
    expect(within(body).getByText("Northline Grocers")).toBeInTheDocument();
    expect(within(body).getByText("INV2610000812")).toBeInTheDocument();
    expect(within(body).getByText("An invoice you may not read")).toBeInTheDocument();
    expect(
      within(section("Reconciliation queue")).getByText(
        "It has never been in the reconciliation queue.",
      ),
    ).toBeInTheDocument();

    fireEvent.click(within(body).getByRole("button", { name: /INV2610000812/u }));
    expect(await screen.findByText("Invoice card inv_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to ACH 7731042" }));

    fireEvent.click(await screen.findByRole("button", { name: /Northline Grocers/u }));
    expect(await screen.findByText("Customer card cus_1")).toBeInTheDocument();
  });

  it("leaves out the payment and the queue for a reader who may read neither", async () => {
    seedPermissions({ [Resource.BankReceipt]: Operation.Read });
    service.getCard.mockResolvedValue(
      matchedCard({ payment: null, paymentHidden: true, workItem: null, workItemShown: false }),
    );
    const { container } = draw("brcpt_2");

    expect(container.querySelectorAll(".dk-rec-skel .dk-rec-sec")).toHaveLength(2);
    await screen.findByRole("heading", { name: "ACH 7731042", level: 2 });
    expect(screen.queryByRole("heading", { name: "The payment it was matched to" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Reconciliation queue" })).toBeNull();
    expect(
      screen.getByText("It is matched to a customer payment you may not read."),
    ).toBeInTheDocument();
  });

  it("lists what an unapplied receipt could pay, and opens an invoice candidate in the card's place", async () => {
    service.getCard.mockResolvedValue(
      exceptionCard({
        candidates: [
          candidate(),
          candidate({
            rank: 2,
            kind: "CustomerPayment",
            invoiceId: "",
            invoiceNumber: "",
            paymentId: "cpay_7",
            paymentReference: "ACH-77",
            standing: "Exact",
            evidence: ["ReferenceExact", "AmountExact"],
            openMinor: 945_000,
            differenceMinor: 0,
          }),
        ],
      }),
    );
    draw();

    const could = section(
      (await screen.findByRole("heading", { name: "What it could pay" })).textContent ?? "",
    );
    expect(within(could).getByText("SEED-INV-0003")).toBeInTheDocument();
    expect(within(could).getByText(`${minorMoney(12_000)} short`)).toBeInTheDocument();
    expect(
      within(could).getByText(`${minorMoney(427_500)} open · The receipt's reference names it`),
    ).toBeInTheDocument();
    expect(within(could).getByText("Premier Farms's payment ACH-77")).toBeInTheDocument();
    expect(within(could).getByText("To the cent")).toBeInTheDocument();

    fireEvent.click(within(could).getByRole("button", { name: /SEED-INV-0003/u }));
    expect(await screen.findByText("Invoice card inv_9")).toBeInTheDocument();
  });

  it("says when nothing names an open item", async () => {
    draw();

    const could = section(
      (await screen.findByRole("heading", { name: "What it could pay" })).textContent ?? "",
    );
    expect(
      within(could).getByText(
        "Nothing the receipt carries names an open invoice or a posted payment.",
      ),
    ).toBeInTheDocument();
  });

  it("says which kinds of candidate the reader may not see, and claims nothing about them", async () => {
    service.getCard.mockResolvedValue(exceptionCard({ candidatesWithheld: ["Invoice"] }));
    draw();

    const could = section(
      (await screen.findByRole("heading", { name: "What it could pay" })).textContent ?? "",
    );
    expect(
      within(could).queryByText(
        "Nothing the receipt carries names an open invoice or a posted payment.",
      ),
    ).toBeNull();
    expect(
      within(could).getByText("You may not read invoices, so invoices it could pay are not shown."),
    ).toBeInTheDocument();
  });

  it("draws no candidates for a matched receipt", async () => {
    draw("brcpt_2");

    await screen.findByRole("heading", { name: "The payment it was matched to" });
    expect(screen.queryByRole("heading", { name: "What it could pay" })).toBeNull();
  });

  it("says the receipt could not be read when the server refuses it", async () => {
    service.getCard.mockRejectedValue(new Error("not found"));
    draw();

    expect(await screen.findByText("This bank receipt could not be read")).toBeInTheDocument();
  });
});
