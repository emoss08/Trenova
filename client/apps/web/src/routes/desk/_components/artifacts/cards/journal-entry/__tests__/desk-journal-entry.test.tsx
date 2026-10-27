import type { DeskJournalEntryCard, JournalCardReversal } from "@/lib/graphql/journal-cards";
import type { JournalReviewResult } from "@/lib/graphql/journal-review";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskJournalEntry } from "../desk-journal-entry";
import pendingFixture from "./je-pending.json";
import postedFixture from "./je-posted.json";
import reversalFixture from "./je-reversal.json";
import reversedFixture from "./je-reversed.json";

const fetchers = vi.hoisted(() => ({
  entry: vi.fn(),
  approve: vi.fn(),
  post: vi.fn(),
  requestReversal: vi.fn(),
}));

vi.mock("@/lib/graphql/journal-cards", () => ({
  fetchDeskJournalEntry: fetchers.entry,
  fetchDeskManualJournal: vi.fn(),
}));
vi.mock("@/lib/graphql/journal-review", () => ({
  approveJournalEntries: fetchers.approve,
  postJournalEntries: fetchers.post,
}));
vi.mock("@/services/api", () => ({
  apiService: { journalReversalService: { create: fetchers.requestReversal } },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));
vi.mock("../../invoice/desk-invoice", () => ({
  DeskInvoice: ({ recordId, backLabel }: { recordId: string; backLabel?: string }) => (
    <p>
      Invoice card {recordId} · {backLabel}
    </p>
  ),
  DeskInvoiceArtifact: () => null,
}));

const NOW_MS = postedFixture.now * 1000;

function card(fixture: { card: unknown }): DeskJournalEntryCard {
  return structuredClone(fixture.card) as DeskJournalEntryCard;
}

const cards: Record<string, () => DeskJournalEntryCard> = {
  je_2610_00418: () => card(postedFixture),
  je_2610_00431: () => card(pendingFixture),
  je_2610_00402: () => card(reversedFixture),
  je_2610_00427: () => card(reversalFixture),
};

function approved(): DeskJournalEntryCard {
  const entry = card(pendingFixture);
  entry.entry.status = "Approved";
  entry.entry.isApproved = true;
  entry.entry.approvedAt = postedFixture.now - 3600;
  entry.approvedBy = { id: "usr_priya", name: "Priya Shah" };
  return entry;
}

function review(changed: boolean, error = ""): JournalReviewResult {
  return {
    changed: changed ? 1 : 0,
    failed: changed ? 0 : 1,
    outcomes: [
      { entryId: "je_2610_00431", entryNumber: "JE-2610-00431", status: "", changed, error },
    ],
  } as JournalReviewResult;
}

/** Signs the test's person in to read entries and, when asked, to review or reverse them. */
function grant({ review = false, reverse = false } = {}) {
  seedPermissions({
    [Resource.JournalEntry]: Operation.Read | (review ? Operation.Approve : 0),
    [Resource.JournalReversal]: Operation.Read | (reverse ? Operation.Create : 0),
    [Resource.Invoice]: Operation.Read,
  });
}

function draw(recordId = "je_2610_00418") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onBack = vi.fn();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskJournalEntry recordId={recordId} onBack={onBack} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { ...view, onBack };
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

function bar(container: HTMLElement): HTMLElement {
  const found = container.querySelector<HTMLElement>(".dk-rec-bar");
  if (!found) throw new Error("No bar");
  return found;
}

function buttons(container: HTMLElement): (string | null)[] {
  return within(bar(container))
    .queryAllByRole("button")
    .map((button) => button.textContent);
}

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW_MS);
  grant();
  fetchers.entry.mockReset().mockImplementation(async (id: string) => {
    const read = cards[id];
    if (!read) throw new Error("Journal entry not found");
    return read();
  });
  fetchers.approve.mockReset().mockResolvedValue(review(true));
  fetchers.post.mockReset().mockResolvedValue(review(true));
  fetchers.requestReversal.mockReset().mockResolvedValue({ id: "jrev_92" });
  vi.mocked(toast.success).mockReset();
  vi.mocked(toast.error).mockReset();
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskJournalEntry", () => {
  it("holds the card's outline while the entry is read, then draws what it booked", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the journal entry…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-jrnl")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$3,612.50each side · balanced",
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Posted");
    expect(container.querySelector(".dk-rec-period")).toHaveTextContent(
      "October 2026Open for posting",
    );
    expect(fetchers.entry).toHaveBeenCalledWith("je_2610_00418", expect.anything());
  });

  it("writes the lines into the book, debits first, and says what they do to the books", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    const journal = section("The journal");
    expect(within(journal).getByText("Debits equal credits")).toBeInTheDocument();
    const rows = within(journal)
      .getByRole("table", { name: "The entry's lines" })
      .querySelectorAll("tbody tr");
    expect(Array.from(rows, (row) => [row.className, row.textContent])).toEqual([
      ["dk-debit", "11200Accounts receivableInvoice INV-2610-0042Acme Foods3,612.50"],
      ["dk-credit", "24000Freight revenueLinehaul, 2 stops2,850.00"],
      ["dk-credit", "34010Fuel surcharge revenueFuel surcharge512.50"],
      ["dk-credit", "44100Accessorial revenueDetentionAcme Plant 2250.00"],
    ]);
    expect(container.querySelector(".dk-rec-jrnl tfoot")).toHaveTextContent(
      "Totals3,612.503,612.50",
    );
    expect(section("What it does to the books")).toHaveTextContent(
      "Both sides of the equation rise by $3,612.50, so the books still balance.",
    );
  });

  it("says when and in which batch it posted, and that Trenova booked it", async () => {
    draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    const stands = section("Where it stands");
    expect(within(stands).getByRole("list", { name: "Journal entry lifecycle" })).toHaveTextContent(
      "RecordedPosted",
    );
    expect(within(stands).getByText(/· batch JB-2610-0031$/u)).toBeInTheDocument();
    expect(
      Array.from(stands.querySelectorAll(".dk-rec-signs dd b"), (name) => name.textContent),
    ).toEqual(["Trenova", "Trenova"]);
    expect(section("Where it came from")).toHaveTextContent(
      "Trenova wrote this entry from its source when the source posted.",
    );
  });

  it("opens the invoice it came from in its place", async () => {
    draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    fireEvent.click(
      within(section("Where it came from")).getByRole("button", { name: /INV-2610-0042/u }),
    );
    expect(
      await screen.findByText("Invoice card inv_2610_0042 · Back to JE-2610-00418"),
    ).toBeInTheDocument();
  });

  it("links the source but opens nothing in place for someone who may not read it", async () => {
    seedPermissions({ [Resource.JournalEntry]: Operation.Read });
    draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    const from = section("Where it came from");
    expect(within(from).queryByRole("button", { name: /INV-2610-0042/u })).toBeNull();
    expect(within(from).getByRole("link", { name: /INV-2610-0042/u })).toHaveAttribute(
      "href",
      expect.stringContaining("inv_2610_0042"),
    );
  });

  it("offers nothing but its page to someone who may only read entries", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    expect(buttons(container)).toEqual([]);
    expect(within(bar(container)).getByRole("link", { name: "Open entry" })).toHaveClass("dk-ink");
  });

  it("asks for a reversal of a posted entry, fixed to the entry, and reads it again", async () => {
    grant({ reverse: true });
    const { container } = draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    expect(buttons(container)).toEqual(["Request reversal"]);
    expect(bar(container)).toHaveTextContent(
      "Reversing posts a mirror entry after someone approves it",
    );
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Request reversal" }));

    const dialog = await screen.findByRole("dialog", {
      name: "Request a reversal of JE-2610-00418",
    });
    expect(within(dialog).queryByLabelText(/Original journal entry/u)).toBeNull();
    fireEvent.change(within(dialog).getByLabelText(/Requested accounting date/u), {
      target: { value: "2026-10-16" },
    });
    fireEvent.change(within(dialog).getByPlaceholderText("e.g., ERROR, DUPLICATE, ADJUSTMENT"), {
      target: { value: " DUPLICATE " },
    });
    fireEvent.change(
      within(dialog).getByPlaceholderText("Provide a detailed reason for the reversal request"),
      {
        target: { value: "Billed twice. " },
      },
    );
    fireEvent.click(within(dialog).getByRole("button", { name: "Request reversal" }));

    await vi.waitFor(() =>
      expect(fetchers.requestReversal).toHaveBeenCalledWith({
        originalJournalEntryId: "je_2610_00418",
        requestedAccountingDate: Date.UTC(2026, 9, 16) / 1000,
        reasonCode: "DUPLICATE",
        reasonText: "Billed twice.",
      }),
    );
    await vi.waitFor(() => expect(fetchers.entry).toHaveBeenCalledTimes(2));
    await vi.waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("shows a reversal that waits on approval in place of the button to ask for one", async () => {
    grant({ reverse: true });
    fetchers.entry.mockImplementation(async () => {
      const read = card(postedFixture);
      read.reversal = {
        id: "jrev_92",
        status: "PendingApproval",
        reasonCode: "WRONG_ACCOUNT",
        reasonText: "Fuel surcharge went to freight revenue.",
        requestedAccountingDate: 1792108800,
        rejectionReason: null,
        cancelReason: null,
        createdAt: 1792081200,
        approvedAt: null,
        postedAt: null,
      } as JournalCardReversal;
      read.reversalRequestedBy = { id: "usr_dana", name: "Dana Ruiz" };
      return read;
    });
    const { container } = draw();
    await screen.findByRole("heading", { name: "Invoice INV-2610-0042 posted", level: 2 });

    expect(section("Its reversal")).toHaveTextContent(
      /^Its reversalDana Ruiz asked for a reversal on .+; it waits on approval/u,
    );
    expect(section("Its reversal")).toHaveTextContent("Fuel surcharge went to freight revenue.");
    expect(buttons(container)).toEqual([]);
  });

  it("brackets a reversed entry with its reversal and opens the reversal in place", async () => {
    grant({ reverse: true });
    const { container } = draw("je_2610_00402");
    await screen.findByRole("heading", { name: "Invoice INV-2610-0038 posted", level: 2 });

    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$1,840.00");
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Reversed");
    const reversal = section("Its reversal");
    expect(within(reversal).getByText("Nets to zero")).toBeInTheDocument();
    expect(reversal).toHaveTextContent(
      "DUPLICATEBilled twice; voided and rebilled as INV-2610-0051.",
    );
    expect(section("Where it stood").querySelector(".dk-rec-track")).toHaveClass("dk-off");
    expect(buttons(container)).toEqual([]);
    expect(bar(container)).toHaveTextContent(
      "A reversed entry stays in the ledger as it was posted",
    );

    fireEvent.click(within(reversal).getByRole("button", { name: /JE-2610-00427/u }));
    expect(
      await screen.findByRole("heading", { name: "Reversal of JE-2610-00402", level: 2 }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to JE-2610-00402" }));
    expect(
      await screen.findByRole("heading", { name: "Invoice INV-2610-0038 posted", level: 2 }),
    ).toBeInTheDocument();
  });

  it("approves an entry waiting on approval and reads it again", async () => {
    grant({ review: true });
    const { container } = draw("je_2610_00431");
    await screen.findByRole("heading", { name: "Worker settlement DS-2610-0147 posted", level: 2 });

    expect(bar(container)).toHaveTextContent("Approving lets it post to October 2026");
    expect(buttons(container)).toEqual(["Approve"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve" }));

    await vi.waitFor(() => expect(fetchers.approve).toHaveBeenCalledWith(["je_2610_00431"]));
    await vi.waitFor(() => expect(fetchers.entry).toHaveBeenCalledTimes(2));
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("says why the review refused the entry instead of claiming it changed", async () => {
    grant({ review: true });
    vi.spyOn(console, "error").mockImplementation(() => {});
    fetchers.approve.mockResolvedValue(review(false, "You cannot approve an entry you booked"));
    const { container } = draw("je_2610_00431");
    await screen.findByRole("heading", { name: "Worker settlement DS-2610-0147 posted", level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve" }));

    await vi.waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith("Error", {
        description: "You cannot approve an entry you booked",
      }),
    );
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("posts an approved entry to the period it lands in", async () => {
    grant({ review: true });
    fetchers.entry.mockResolvedValue(approved());
    const { container } = draw("je_2610_00431");
    await screen.findByRole("heading", { name: "Worker settlement DS-2610-0147 posted", level: 2 });

    expect(bar(container)).toHaveTextContent("It posts to October 2026 on Oct 13, 2026");
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Post to October 2026" }));

    await vi.waitFor(() => expect(fetchers.post).toHaveBeenCalledWith(["je_2610_00431"]));
    expect(fetchers.approve).not.toHaveBeenCalled();
  });

  it("holds the post button with the reason the period refuses the entry", async () => {
    grant({ review: true });
    fetchers.entry.mockImplementation(async () => {
      const read = approved();
      read.posting = {
        period: null,
        accountingDate: null,
        refusal: "September 2026 is closed and your policy refuses late postings",
      };
      return read;
    });
    const { container } = draw("je_2610_00431");
    await screen.findByRole("heading", { name: "Worker settlement DS-2610-0147 posted", level: 2 });

    const postButton = within(bar(container)).getByRole("button", { name: "Post" });
    expect(postButton).toBeDisabled();
    expect(postButton).toHaveAttribute(
      "aria-description",
      "September 2026 is closed and your policy refuses late postings",
    );
    expect(bar(container)).toHaveTextContent(
      "September 2026 is closed and your policy refuses late postings",
    );
    fireEvent.click(postButton);
    expect(fetchers.post).not.toHaveBeenCalled();
  });

  it("offers no review step to someone who may not approve entries", async () => {
    const { container } = draw("je_2610_00431");
    await screen.findByRole("heading", { name: "Worker settlement DS-2610-0147 posted", level: 2 });

    expect(buttons(container)).toEqual([]);
  });

  it("says the entry could not be read, with the way back", async () => {
    const { onBack } = draw("je_2610_00001");

    expect(await screen.findByText("This journal entry could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });
});
