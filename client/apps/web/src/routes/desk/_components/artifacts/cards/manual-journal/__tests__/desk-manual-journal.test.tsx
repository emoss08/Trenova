import type { DeskManualJournalCard } from "@/lib/graphql/journal-cards";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { toast } from "sonner";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskManualJournal } from "../desk-manual-journal";
import draftFixture from "./mj-draft.json";
import pendingFixture from "./mj-pending.json";
import postedFixture from "./mj-posted.json";
import rejectedFixture from "./mj-rejected.json";

const fetchers = vi.hoisted(() => ({
  journal: vi.fn(),
  submit: vi.fn(),
  approve: vi.fn(),
  post: vi.fn(),
  reject: vi.fn(),
  cancel: vi.fn(),
}));

vi.mock("@/lib/graphql/journal-cards", () => ({
  fetchDeskJournalEntry: vi.fn(),
  fetchDeskManualJournal: fetchers.journal,
}));
vi.mock("@/services/api", () => ({
  apiService: {
    manualJournalService: {
      submit: fetchers.submit,
      approve: fetchers.approve,
      post: fetchers.post,
      reject: fetchers.reject,
      cancel: fetchers.cancel,
    },
  },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("../../journal-entry/desk-journal-entry", () => ({
  DeskJournalEntry: ({ recordId, backLabel }: { recordId: string; backLabel?: string }) => (
    <p>
      Journal entry card {recordId} · {backLabel}
    </p>
  ),
  DeskJournalEntryArtifact: () => null,
}));

const NOW_MS = pendingFixture.now * 1000;

function card(fixture: { card: unknown }): DeskManualJournalCard {
  return structuredClone(fixture.card) as DeskManualJournalCard;
}

/** Signs the test's person in to read manual journals and take the steps given. */
function grant(operations = 0) {
  seedPermissions({
    [Resource.ManualJournal]: Operation.Read | operations,
    [Resource.JournalEntry]: Operation.Read,
  });
}

const EVERY_STEP = Operation.Submit | Operation.Approve | Operation.Reject | Operation.Cancel;

function draw() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onBack = vi.fn();
  const view = render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskManualJournal recordId="mjr_2610_0007" onBack={onBack} />
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

const PENDING_TITLE = "Accrue October fuel not yet invoiced";
const DRAFT_TITLE = "Reclassify September tolls to repairs";

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW_MS);
  grant();
  fetchers.journal.mockReset().mockResolvedValue(card(pendingFixture));
  for (const step of [
    fetchers.submit,
    fetchers.approve,
    fetchers.post,
    fetchers.reject,
    fetchers.cancel,
  ]) {
    step.mockReset().mockResolvedValue({ id: "mjr_2610_0007", status: "Posted" });
  }
  vi.mocked(toast.success).mockReset();
  vi.mocked(toast.error).mockReset();
});

afterEach(() => {
  clearPermissions();
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("DeskManualJournal", () => {
  it("holds the card's outline while the journal is read, then draws what it books", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the manual journal…");
    expect(container.querySelector(".dk-rec-skel .dk-mj-why")).not.toBeNull();

    expect(
      await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 }),
    ).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "$24,535.50each side · balanced",
    );
    expect(container.querySelector(".dk-rec-st")).toHaveTextContent("Waiting on approval");
    expect(fetchers.journal).toHaveBeenCalledWith("mjr_2610_0007", expect.anything());
  });

  it("gives the requester's reason, the lines, what they do and where it will post", async () => {
    draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    expect(section("Why")).toHaveTextContent(
      "Pilot and Love's send October's statements after month end.",
    );
    expect(section("Why")).toHaveTextContent("Dana Ruiz");
    expect(
      within(section("The journal")).getByRole("table", { name: "The journal's lines" }),
    ).toHaveTextContent("Totals24,535.5024,535.50");
    expect(section("What it does to the books")).toHaveTextContent(
      "Assets don't move, and the other side nets to zero, so the books still balance.",
    );
    expect(section("Where it will post")).toHaveTextContent(
      "October 2026Open for postingPosts on Oct 31, 2026, the date asked for",
    );
  });

  it("offers nothing but its page to someone who may only read journals", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    expect(buttons(container)).toEqual([]);
    expect(within(bar(container)).getByRole("link", { name: "Open journal" })).toHaveClass(
      "dk-ink",
    );
  });

  it("approves and posts a submitted journal in one step, in that order", async () => {
    grant(EVERY_STEP);
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    expect(buttons(container)).toEqual(["Reject", "Cancel", "Approve and post"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve and post" }));

    await vi.waitFor(() => expect(fetchers.post).toHaveBeenCalledWith("mjr_2610_0007"));
    expect(fetchers.approve).toHaveBeenCalledWith("mjr_2610_0007");
    expect(fetchers.approve.mock.invocationCallOrder[0]).toBeLessThan(
      fetchers.post.mock.invocationCallOrder[0]!,
    );
    await vi.waitFor(() => expect(fetchers.journal).toHaveBeenCalledTimes(2));
    expect(toast.success).toHaveBeenCalledWith("MJ-2610-0007 approved and posted");
  });

  it("reads the journal again when it was approved but could not post", async () => {
    grant(EVERY_STEP);
    vi.spyOn(console, "error").mockImplementation(() => {});
    fetchers.post.mockRejectedValue(new Error("October 2026 was closed a moment ago"));
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve and post" }));

    await vi.waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith("Error", {
        description: "October 2026 was closed a moment ago",
      }),
    );
    await vi.waitFor(() => expect(fetchers.journal).toHaveBeenCalledTimes(2));
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("does not post a journal the server would not approve", async () => {
    grant(EVERY_STEP);
    vi.spyOn(console, "error").mockImplementation(() => {});
    fetchers.approve.mockRejectedValue(new Error("You cannot approve your own journal"));
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Approve and post" }));

    await vi.waitFor(() => expect(toast.error).toHaveBeenCalled());
    expect(fetchers.post).not.toHaveBeenCalled();
  });

  it("posts a journal approved already, and only posts it", async () => {
    grant(EVERY_STEP);
    fetchers.journal.mockImplementation(async () => {
      const read = card(pendingFixture);
      read.journal.status = "Approved";
      read.journal.approvedAt = pendingFixture.now - 3600;
      read.approvedBy = { id: "usr_priya", name: "Priya Shah" };
      return read;
    });
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    expect(buttons(container)).toEqual(["Cancel", "Post"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Post" }));
    await vi.waitFor(() => expect(fetchers.post).toHaveBeenCalledWith("mjr_2610_0007"));
    expect(fetchers.approve).not.toHaveBeenCalled();
  });

  it("holds approve and post with the reason the journal cannot post", async () => {
    grant(EVERY_STEP);
    const refusal = "September 2026 is closed and your policy refuses late postings";
    fetchers.journal.mockImplementation(async () => {
      const read = card(pendingFixture);
      read.posting = { period: null, accountingDate: null, refusal };
      return read;
    });
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    const step = within(bar(container)).getByRole("button", { name: "Approve and post" });
    expect(step).toBeDisabled();
    expect(step).toHaveAttribute("aria-description", refusal);
    expect(section("Where it will post")).toHaveTextContent(refusal);
  });

  it("rejects a journal only with a reason, and passes the reason on", async () => {
    grant(Operation.Reject);
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    expect(buttons(container)).toEqual(["Reject"]);
    fireEvent.click(within(bar(container)).getByRole("button", { name: "Reject" }));
    const dialog = await screen.findByRole("dialog", { name: "Reject MJ-2610-0007" });

    fireEvent.click(within(dialog).getByRole("button", { name: "Reject" }));
    expect(await within(dialog).findByText("Say why")).toBeInTheDocument();
    expect(fetchers.reject).not.toHaveBeenCalled();

    fireEvent.change(within(dialog).getByPlaceholderText("What needs to change"), {
      target: { value: "  Wait for the statements; accrue from the fuel card feed.  " },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Reject" }));
    await vi.waitFor(() =>
      expect(fetchers.reject).toHaveBeenCalledWith(
        "mjr_2610_0007",
        "Wait for the statements; accrue from the fuel card feed.",
      ),
    );
    await vi.waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(toast.success).toHaveBeenCalledWith("MJ-2610-0007 is rejected");
  });

  it("refuses a reason longer than the server keeps", async () => {
    grant(Operation.Cancel);
    const { container } = draw();
    await screen.findByRole("heading", { name: PENDING_TITLE, level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Cancel" }));
    const dialog = await screen.findByRole("dialog", { name: "Cancel MJ-2610-0007" });
    fireEvent.change(within(dialog).getByPlaceholderText("Why it is no longer needed"), {
      target: { value: "x".repeat(501) },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel journal" }));

    expect(
      await within(dialog).findByText("Keep the reason under 500 characters"),
    ).toBeInTheDocument();
    expect(fetchers.cancel).not.toHaveBeenCalled();
  });

  it("holds a draft that does not balance, says by how much and where it will land", async () => {
    grant(EVERY_STEP);
    fetchers.journal.mockResolvedValue(card(draftFixture));
    const { container } = draw();
    await screen.findByRole("heading", { name: DRAFT_TITLE, level: 2 });

    expect(container.querySelector(".dk-rec-amt")).toHaveTextContent(
      "Off by $40.00debits exceed credits",
    );
    expect(container.querySelector(".dk-rec-jrnl .dk-off")).toHaveTextContent(
      "Credits are short by40.00",
    );
    expect(section("What it does to the books")).toHaveTextContent(
      "Debits and credits differ by $40.00, so the books won't balance until the lines do.",
    );
    expect(section("Where it will post")).toHaveTextContent(
      "September 2026 takes no more postings, and your accounting policy books late entries on the first open day.",
    );
    expect(buttons(container)).toEqual(["Cancel", "Submit for approval"]);
    const submit = within(bar(container)).getByRole("button", { name: "Submit for approval" });
    expect(submit).toBeDisabled();
    expect(submit).toHaveAttribute(
      "aria-description",
      "Balance the lines on its page before submitting",
    );
    expect(bar(container)).toHaveTextContent("Balance the lines on its page");
  });

  it("submits a balanced draft and says when it went straight to approved", async () => {
    grant(Operation.Submit);
    fetchers.journal.mockImplementation(async () => {
      const read = card(draftFixture);
      read.lines[1]!.creditAmount = 126000;
      return read;
    });
    fetchers.submit.mockResolvedValue({ id: "mjr_2610_0009", status: "Approved" });
    const { container } = draw();
    await screen.findByRole("heading", { name: DRAFT_TITLE, level: 2 });

    fireEvent.click(within(bar(container)).getByRole("button", { name: "Submit for approval" }));
    await vi.waitFor(() => expect(fetchers.submit).toHaveBeenCalledWith("mjr_2610_0009"));
    await vi.waitFor(() =>
      expect(toast.success).toHaveBeenCalledWith("MJ-2610-0009 submitted and approved"),
    );
  });

  it("names the entry a posted journal became and opens it in place", async () => {
    grant(EVERY_STEP);
    fetchers.journal.mockResolvedValue(card(postedFixture));
    const { container } = draw();
    await screen.findByRole("heading", { name: "Write off Harbor Freight short pay", level: 2 });

    expect(buttons(container)).toEqual([]);
    expect(section("What it became")).toHaveTextContent(
      /^What it becameJEJE-2610-00422Journal entry · posted .+ · \$312\.00$/u,
    );
    expect(screen.queryByRole("heading", { name: "Where it will post" })).toBeNull();
    fireEvent.click(
      within(section("What it became")).getByRole("button", { name: /JE-2610-00422/u }),
    );
    expect(
      await screen.findByText("Journal entry card je_2610_00422 · Back to MJ-2610-0004"),
    ).toBeInTheDocument();
  });

  it("keeps why a journal was rejected and strikes what it would have booked", async () => {
    grant(EVERY_STEP);
    fetchers.journal.mockResolvedValue(card(rejectedFixture));
    const { container } = draw();
    await screen.findByRole("heading", { name: "Write off Acme detention dispute", level: 2 });

    expect(section("Why it was rejected")).toHaveTextContent(
      "Waive it under the detention policy instead of a manual write-off, so the stop's record shows why.",
    );
    expect(container.querySelector(".dk-rec-amt s")).toHaveTextContent("$250.00");
    expect(section("Where it stood").querySelector(".dk-rec-track")).toHaveClass("dk-off");
    expect(buttons(container)).toEqual([]);
  });

  it("says the journal could not be read, with the way back", async () => {
    fetchers.journal.mockRejectedValue(new Error("Manual journal not found"));
    const { onBack } = draw();

    expect(await screen.findByText("This manual journal could not be read")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalled();
  });
});
