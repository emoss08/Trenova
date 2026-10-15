import type { AccountingSyncObjectState } from "@/lib/graphql/accounting-sync-ledger";
import type { ActivityEntry } from "@/lib/graphql/record-activity";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskInvoice, DeskInvoiceArtifact } from "../desk-invoice";
import {
  consolidatedDraft,
  creditMemo,
  DAY,
  HOUR,
  LONG_TEMPLATE,
  postedInvoice,
  voidedInvoice,
} from "./fixtures";

const fetchers = vi.hoisted(() => ({
  invoice: vi.fn(),
  activity: vi.fn(),
  shipments: vi.fn(),
  syncStates: vi.fn(),
  attempts: vi.fn(),
  plan: vi.fn(),
  send: vi.fn(),
  generatePdf: vi.fn(),
  postPreview: vi.fn(),
  post: vi.fn(),
  sendEdi: vi.fn(),
  voidInvoice: vi.fn(),
}));

vi.mock("@/lib/graphql/invoice-card", () => ({ fetchDeskInvoice: fetchers.invoice }));
vi.mock("@/lib/graphql/record-activity", () => ({ fetchRecordActivity: fetchers.activity }));
vi.mock("@/lib/graphql/record-shipments", () => ({ fetchRecordShipments: fetchers.shipments }));
vi.mock("@/lib/graphql/accounting-sync-ledger", () => ({
  fetchAccountingSyncObjectStates: fetchers.syncStates,
  fetchAccountingSyncSummary: vi.fn(),
  fetchAccountingSyncRecord: vi.fn(),
  fetchAccountingSyncAttempts: vi.fn(),
  fetchAccountingBackfills: vi.fn(),
}));
vi.mock("@/services/api", () => ({
  apiService: {
    invoiceService: {
      listEmailAttempts: fetchers.attempts,
      getSendPlan: fetchers.plan,
      send: fetchers.send,
      generatePdf: fetchers.generatePdf,
      getPostPreview: fetchers.postPreview,
      post: fetchers.post,
    },
  },
}));
vi.mock("@/lib/graphql/invoice", () => ({
  sendInvoiceEdi: fetchers.sendEdi,
  voidInvoice: fetchers.voidInvoice,
  fetchInvoiceArContext: vi.fn(),
  fetchInvoicesByShipment: vi.fn(),
}));
// The formula template and EDI message cards are chunks of their own; here each
// stands in as what it was given, with its way back.
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
vi.mock("../../formula-template/desk-formula-template", () => ({
  DeskFormulaTemplate: standIn("Formula template"),
  DeskFormulaTemplateArtifact: () => null,
}));
vi.mock("../../edi-message/desk-edi-message", () => ({
  DeskEdiMessage: standIn("Message"),
  DeskEdiMessageArtifact: () => null,
}));
vi.mock("../../shipment/desk-shipment", () => ({
  DeskShipment: standIn("Shipment"),
  DeskShipmentArtifact: () => null,
}));

const NOW_MS = (DAY.oct14 + 15 * HOUR) * 1000;

const synced: AccountingSyncObjectState = {
  objectType: "Invoice",
  objectId: "inv_412",
  providerName: "QuickBooks Online",
  record: {
    id: "acctsr_1",
    status: "Synced",
    syncedAt: DAY.sep2 + 9 * HOUR,
    nextAttemptAt: null,
    resolution: "",
    errorMessage: "",
    attemptCount: 1,
    skippedReason: "",
    externalDocNumber: "1042",
    externalUrl: "https://app.qbo.intuit.com/app/invoice?txnId=1042",
  },
} as unknown as AccountingSyncObjectState;

const entry = {
  id: "ae_1",
  operation: "update",
  timestamp: DAY.oct9,
  changes: { status: { from: "Posted", to: "Posted" } },
  comment: "Applied credit memo CM-2610-0031",
  principalType: "session_user",
  user: { id: "usr_dana", name: "Dana Ruiz" },
} as ActivityEntry;

const shipment = {
  id: "shp_240193",
  proNumber: "S-240193",
  status: "Invoiced",
  totalChargeAmount: "3666.12",
  moves: [],
} as unknown as RecordShipment;

function draw(props: Partial<Parameters<typeof DeskInvoice>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskInvoice recordId="inv_412" order={["inv_400", "inv_412"]} {...handlers} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, ...handlers, queryClient };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW_MS);
  seedPermissions({
    [Resource.Invoice]: Operation.Read,
    [Resource.Shipment]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
    [Resource.AccountingSync]: Operation.Read,
    [Resource.FormulaTemplate]: Operation.Read,
    [Resource.EDI]: Operation.Read,
  });
  fetchers.invoice.mockReset().mockImplementation(async (id: string) => {
    switch (id) {
      case "inv_412":
        return postedInvoice();
      case "inv_457":
        return consolidatedDraft();
      case "inv_388":
        return voidedInvoice();
      case "inv_cm31":
        return creditMemo();
      default:
        throw new Error("Invoice not found");
    }
  });
  fetchers.activity.mockReset().mockResolvedValue({
    rows: [entry],
    total: 1,
    endCursor: null,
    hasNextPage: false,
  });
  fetchers.shipments.mockReset().mockResolvedValue({
    rows: [shipment],
    total: 1,
    endCursor: null,
    hasNextPage: false,
  });
  fetchers.syncStates.mockReset().mockResolvedValue([synced]);
  fetchers.attempts.mockReset().mockResolvedValue({
    count: 1,
    next: null,
    results: [{ toRecipients: ["ap@acmefoods.com"], sentAt: DAY.sep2 + 9 * HOUR, error: null }],
  });
  fetchers.plan.mockReset().mockResolvedValue({
    recipients: { to: ["ap@tyson.com"], cc: [], bcc: [] },
    errors: [],
  });
  for (const action of [
    fetchers.send,
    fetchers.generatePdf,
    fetchers.postPreview,
    fetchers.post,
    fetchers.sendEdi,
    fetchers.voidInvoice,
  ]) {
    action.mockReset();
  }
});

/** Signs the test's person in to read every section and take these invoice actions. */
function grantInvoice(operations: number) {
  seedPermissions({
    [Resource.Invoice]: Operation.Read | operations,
    [Resource.Shipment]: Operation.Read,
    [Resource.AuditLog]: Operation.Read,
    [Resource.AccountingSync]: Operation.Read,
    [Resource.FormulaTemplate]: Operation.Read,
    [Resource.EDI]: Operation.Read,
  });
}

function bar(container: HTMLElement): HTMLElement {
  const found = container.querySelector<HTMLElement>(".dk-rec-bar");
  if (!found) throw new Error("No bar");
  return found;
}

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskInvoice", () => {
  it("holds the card's outline while the invoice is read, then draws who it bills and what is owed", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the invoice…");
    expect(container.querySelector(".dk-rec-skel .dk-inv-slip")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-inv-bill")).not.toBeNull();
    expect(container.querySelector(".dk-rec-skel .dk-inv-went")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$591.12still owed of $3,666.12",
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("12 days past due");

    const owed = section("What is owed");
    expect(within(owed).getByRole("img")).toHaveAccessibleName(
      "Paid $2,850.00, Credited $225.00, In dispute $185.00, Open, not disputed $406.12",
    );
    expect(within(owed).getByText("12 days past due", { selector: "b" })).toBeInTheDocument();
  });

  it("names the formula template that rated the freight and opens its card in the invoice's place", async () => {
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    const charges = section("Charges");
    expect(
      within(charges)
        .getAllByRole("row")
        .map((row) => row.textContent),
    ).toEqual([
      "ChargeRated byAmount",
      `Linehaul$3.75 base rateƒ${LONG_TEMPLATE}$2,850.00`,
      "Fuel surchargeFSC14.25% of $2,850.00$406.12",
      "DetentionDET3 hours × $75.00$225.00",
      "LumperLUMPFlat$185.00",
    ]);

    fireEvent.click(
      within(charges).getByRole("button", { name: `Open formula template ${LONG_TEMPLATE}` }),
    );
    expect(await screen.findByText("Formula template card ft_permile")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to INV-2610-0412" }));
    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
  });

  it("reads a template's name only, without a way in, for someone who may not read templates", async () => {
    seedPermissions({ [Resource.Invoice]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    const charges = section("Charges");
    expect(within(charges).queryByRole("button", { name: /Open formula template/u })).toBeNull();
    expect(within(charges).getByText(LONG_TEMPLATE)).toBeInTheDocument();
  });

  it("cuts a long line to fit and lets it be read in full only when it was cut", async () => {
    const width = vi
      .spyOn(HTMLElement.prototype, "scrollWidth", "get")
      .mockImplementation(function (this: HTMLElement) {
        return (this.textContent ?? "").length > 35 ? 400 : 100;
      });
    vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockReturnValue(200);
    fetchers.invoice.mockResolvedValue(consolidatedDraft());
    draw({ recordId: "inv_457" });
    await screen.findByRole("heading", { name: "Tyson Foods", level: 2 });

    const why = screen.getByText("Tyson Foods has no 210 document profile.");
    expect(why).toHaveClass("dk-inv-clip", "dk-inv-why");
    expect(why).toHaveAttribute("tabindex", "0");
    const short = await screen.findByText("Goes to ap@tyson.com when sent");
    expect(short).not.toHaveAttribute("tabindex");
    width.mockRestore();
  });

  it("reads the email it went by, its 210 with a way into the message, and the books it is in", async () => {
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    const went = section("Where it went");
    expect(await within(went).findByText("In QuickBooks Online")).toBeInTheDocument();
    expect(within(went).getByRole("link", { name: "Open in QuickBooks Online" })).toHaveAttribute(
      "href",
      "https://app.qbo.intuit.com/app/invoice?txnId=1042",
    );
    expect(fetchers.attempts).toHaveBeenCalledWith("inv_412", { limit: 1 });
    expect(fetchers.plan).not.toHaveBeenCalled();
    expect(await within(went).findByText(/to ap@acmefoods\.com$/u)).toBeInTheDocument();

    fireEvent.click(within(went).getByRole("button", { name: "Open message" }));
    expect(await screen.findByText("Message card edim_210")).toBeInTheDocument();
  });

  it("leaves out what the person may not read and never asks for it", async () => {
    seedPermissions({ [Resource.Invoice]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    expect(screen.queryByRole("heading", { name: "Shipment it bills" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Activity" })).toBeNull();
    expect(within(section("Where it went")).queryByText("Accounting")).toBeNull();
    expect(fetchers.shipments).not.toHaveBeenCalled();
    expect(fetchers.activity).not.toHaveBeenCalled();
    expect(fetchers.syncStates).not.toHaveBeenCalled();
  });

  it("counts the days a draft's terms have run while it waits, and reads where it will go from its plan", async () => {
    fetchers.invoice.mockResolvedValue(consolidatedDraft());
    draw({ recordId: "inv_457" });
    await screen.findByRole("heading", { name: "Tyson Foods", level: 2 });

    const waiting = section("Waiting to post");
    expect(within(waiting).getByText("13 days unposted")).toBeInTheDocument();
    expect(waiting).toHaveTextContent(
      "Its terms count from the invoice date, so 13 of its 45 days have run while it waits to be posted.",
    );
    expect(screen.queryByRole("heading", { name: "What is owed" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Payments and credits" })).toBeNull();

    const going = section("Where it will go");
    expect(within(going).getByText("Cannot send")).toBeInTheDocument();
    expect(fetchers.plan).toHaveBeenCalledWith("inv_457");
    expect(fetchers.attempts).not.toHaveBeenCalled();
    expect(fetchers.syncStates).not.toHaveBeenCalled();
    expect(await within(going).findByText("Goes to ap@tyson.com when sent")).toBeInTheDocument();
  });

  it("says why a voided invoice was voided and takes its balance to nothing", async () => {
    fetchers.invoice.mockResolvedValue(voidedInvoice());
    const { container } = draw({ recordId: "inv_388" });
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$2,140.00");
    const why = section("Why it was voided");
    expect(why).toHaveTextContent("Billed to Acme, but the consignee pays on this lane.");
    expect(why).toHaveTextContent("Back in the billing queue to bill again");
    expect(within(section("Charges")).getByText("Reversed")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Payments and credits" })).toBeNull();
  });

  it("opens the invoice a credit memo settled in the memo's place, and comes back to the memo", async () => {
    fetchers.invoice.mockImplementation(async (id: string) =>
      id === "inv_cm31" ? creditMemo() : postedInvoice(),
    );
    const { container } = draw({ recordId: "inv_cm31" });
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$175.00credit left of $400.00",
    );
    expect(container.querySelector(".dk-inv-slip")).toHaveTextContent("CR");
    const applied = section("Applied to");
    const rows = within(applied).getAllByRole("listitem");
    expect(rows.map((row) => row.classList.contains("dk-undone"))).toEqual([false, true]);

    fireEvent.click(within(rows[0]!).getByRole("button"));
    expect(
      await screen.findByText("12 days past due", { selector: ".dk-rec-st" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to CM-2610-0031" }));
    expect(
      await screen.findByRole("heading", { name: "Applied to", level: 3 }),
    ).toBeInTheDocument();
  });

  it("says the invoice could not be read, with the way back", async () => {
    fetchers.invoice.mockRejectedValue(new Error("Invoice not found"));
    const { onBack } = draw({ recordId: "inv_gone" });

    expect(await screen.findByText("This invoice could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });

  it("draws the same card as an artifact, with nothing to step through and nowhere to go back to", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskInvoiceArtifact
            artifact={{} as Parameters<typeof DeskInvoiceArtifact>[0]["artifact"]}
            recordId="inv_412"
            fallback={null}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("heading", { name: "Acme Foods", level: 2 }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back to the table" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Next invoice" })).toBeNull();
    expect(screen.getByRole("link", { name: "Open invoice" })).toHaveAttribute(
      "href",
      expect.stringContaining("inv_412"),
    );
  });
});

describe("DeskInvoice actions", () => {
  it("offers no action to someone who may only read the invoice", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
    const went = section("Where it went");
    expect(within(went).queryByRole("button", { name: "Resend" })).toBeNull();
    expect(within(went).queryByRole("button", { name: "Resend 210" })).toBeNull();
  });

  it("posts a draft only after showing what posting it books", async () => {
    grantInvoice(Operation.Update);
    fetchers.invoice.mockResolvedValue(consolidatedDraft());
    fetchers.postPreview.mockResolvedValue({
      invoiceId: "inv_457",
      number: "INV-2610-0457",
      billToName: "Tyson Foods",
      currencyCode: "USD",
      totalMinor: 1_842_000,
      refusals: [],
      alreadyPosted: false,
      journal: null,
      shipmentsBilled: [],
      queueItemPosted: false,
      accountingSync: [],
      edi: null,
    });
    fetchers.post.mockResolvedValue({ id: "inv_457", number: "INV-2610-0457" });
    const { container } = draw({ recordId: "inv_457" });
    await screen.findByRole("heading", { name: "Tyson Foods", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Post $18,420.00" }));
    const dialog = await screen.findByRole("dialog");
    expect(fetchers.postPreview).toHaveBeenCalledWith("inv_457");
    expect(fetchers.post).not.toHaveBeenCalled();

    fireEvent.click(await within(dialog).findByRole("button", { name: "Post invoice" }));
    await vi.waitFor(() => expect(fetchers.post).toHaveBeenCalledWith("inv_457"));
  });

  it("emails a posted invoice again and sends its 210 again for someone who may", async () => {
    grantInvoice(Operation.Update | Operation.Submit);
    fetchers.send.mockResolvedValue({});
    fetchers.sendEdi.mockResolvedValue({ status: "Queued" });
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });
    const went = section("Where it went");

    const resend = await within(went).findByRole("button", { name: "Resend" });
    await vi.waitFor(() => expect(resend).toBeEnabled());
    fireEvent.click(resend);
    await vi.waitFor(() => expect(fetchers.send).toHaveBeenCalledWith("inv_412"));
    await vi.waitFor(() => expect(fetchers.invoice).toHaveBeenCalledTimes(2));

    fireEvent.click(within(went).getByRole("button", { name: "Resend 210" }));
    await vi.waitFor(() => expect(fetchers.sendEdi).toHaveBeenCalledWith("inv_412", true));
    await vi.waitFor(() => expect(fetchers.invoice).toHaveBeenCalledTimes(3));
  });

  it("holds the email back and says why while the send plan has an error", async () => {
    grantInvoice(Operation.Submit);
    fetchers.plan.mockResolvedValue({
      recipients: { to: [], cc: [], bcc: [] },
      errors: ["Acme Foods has no billing email"],
    });
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    const resend = within(section("Where it went")).getByRole("button", { name: "Resend" });
    await vi.waitFor(() =>
      expect(resend).toHaveAttribute("aria-description", "Acme Foods has no billing email"),
    );
    expect(resend).toBeDisabled();
  });

  it("makes the invoice's PDF first when it has none to send", async () => {
    grantInvoice(Operation.Update | Operation.Submit);
    fetchers.invoice.mockResolvedValue(postedInvoice({ pdfDocumentId: null }));
    fetchers.generatePdf.mockResolvedValue(undefined);
    draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });
    const went = section("Where it went");

    expect(within(went).queryByRole("button", { name: "Resend" })).toBeNull();
    fireEvent.click(within(went).getByRole("button", { name: "Generate PDF" }));
    await vi.waitFor(() => expect(fetchers.generatePdf).toHaveBeenCalledWith("inv_412"));
  });

  it("will not void an invoice while cash is applied to it, and says so", async () => {
    grantInvoice(Operation.Cancel);
    const { container } = draw();
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    const voiding = within(bar(container)).getByRole("button", { name: "Void" });
    expect(voiding).toBeDisabled();
    expect(voiding).toHaveAttribute(
      "aria-description",
      "Unapply the customer payments and credit memos on this invoice before voiding it.",
    );
  });

  it("voids an invoice with nothing applied to it through the reason it is given", async () => {
    grantInvoice(Operation.Cancel);
    fetchers.invoice.mockResolvedValue(consolidatedDraft());
    const { container } = draw({ recordId: "inv_457" });
    await screen.findByRole("heading", { name: "Tyson Foods", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Void" }));
    expect(
      await screen.findByRole("heading", { name: "Void invoice INV-2610-0457" }),
    ).toBeInTheDocument();
    expect(fetchers.voidInvoice).not.toHaveBeenCalled();
  });

  it("offers nothing to do to a voided invoice", async () => {
    grantInvoice(Operation.Update | Operation.Submit | Operation.Cancel);
    fetchers.invoice.mockResolvedValue(voidedInvoice());
    const { container } = draw({ recordId: "inv_388" });
    await screen.findByRole("heading", { name: "Acme Foods", level: 2 });

    expect(within(bar(container)).queryAllByRole("button")).toHaveLength(0);
    expect(fetchers.plan).not.toHaveBeenCalled();
  });
});
