import { recordPath } from "@/config/record-links";
import type { JournalReviewLabels } from "@/hooks/use-journal-review-labels";
import type { JournalReviewRow, JournalReviewSummary } from "@/lib/graphql/journal-review";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter, type UrlUpdateEvent } from "nuqs/adapters/testing";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getJournalReviewColumns } from "../_components/review-columns";
import {
  activeJournalReviewBucket,
  journalReviewBucketFilter,
  journalReviewPhase,
} from "../_components/review-filters";
import { JournalReviewSummaryStrip } from "../_components/review-summary";
import { JournalReviewPage } from "../page";

const mocks = vi.hoisted(() => ({
  fetchJournalReviewSummary: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/journal-review", () => ({
  fetchJournalReviewSummary: mocks.fetchJournalReviewSummary,
  approveJournalEntries: vi.fn(),
  postJournalEntries: vi.fn(),
  JOURNAL_REVIEW_TABLE_KEY: "journal-review",
  journalReviewTableGraphQLConfig: {},
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("../_components/review-table", () => ({
  default: () => <div>journal review table</div>,
}));

const APPROVE = `${Resource.JournalEntry}:${Operation.Approve}`;

function summary(overrides: Partial<JournalReviewSummary> = {}): JournalReviewSummary {
  return {
    awaitingApproval: 4,
    readyToPost: 2,
    oldestAccountingDate: 1_790_208_000,
    postingMode: "Manual",
    requiresApproval: true,
    ...overrides,
  };
}

function renderWith(node: ReactNode, searchParams = "") {
  const urls: UrlUpdateEvent[] = [];
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const view = render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter
          searchParams={searchParams}
          hasMemory
          onUrlUpdate={(event) => urls.push(event)}
        >
          {node}
        </NuqsTestingAdapter>
      </QueryClientProvider>
    </MemoryRouter>,
  );
  return { ...view, urls };
}

const t = ((template: string, ...args: unknown[]) =>
  template.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

const labels: JournalReviewLabels = {
  status: { Pending: "Awaiting approval", Approved: "Ready to post" },
  source: (referenceType) =>
    referenceType === "CreditMemoPosted" ? "Credit memo posted" : referenceType,
};

describe("journal review buckets", () => {
  it("round-trips each bucket and gives each status its phase", () => {
    expect(activeJournalReviewBucket(journalReviewBucketFilter("awaiting"))).toBe("awaiting");
    expect(activeJournalReviewBucket(journalReviewBucketFilter("ready"))).toBe("ready");
    expect(activeJournalReviewBucket([])).toBeNull();
    expect(journalReviewPhase("Pending")).toBe("awaiting");
    expect(journalReviewPhase("Approved")).toBe("queued");
  });
});

describe("journal review summary", () => {
  it("counts the queue and filters the table to entries awaiting approval", async () => {
    const { urls } = renderWith(<JournalReviewSummaryStrip summary={summary()} />);

    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();

    await userEvent.click(screen.getByText("Awaiting approval"));
    const last = urls.at(-1);
    expect(last?.queryString).toContain("Pending");
  });
});

describe("journal review page", () => {
  beforeEach(() => {
    mocks.fetchJournalReviewSummary.mockReset();
    mocks.granted.clear();
  });

  it("explains entries left over from manual posting once posting is automatic", async () => {
    mocks.granted.add(APPROVE);
    mocks.fetchJournalReviewSummary.mockResolvedValue(
      summary({ postingMode: "Automatic", requiresApproval: false }),
    );

    renderWith(<JournalReviewPage />);

    expect(
      await screen.findByText(/These were written while posting was manual/),
    ).toBeInTheDocument();
    expect(screen.queryByText(/needs the Journal Entry approve permission/)).toBeNull();
  });

  it("says nothing about automatic posting in manual mode", async () => {
    mocks.granted.add(APPROVE);
    mocks.fetchJournalReviewSummary.mockResolvedValue(summary());

    renderWith(<JournalReviewPage />);

    expect(await screen.findByText("Ready to post")).toBeInTheDocument();
    expect(screen.queryByText(/These were written while posting was manual/)).toBeNull();
  });

  it("tells a person without the approve permission why they cannot act", async () => {
    mocks.fetchJournalReviewSummary.mockResolvedValue(summary());

    renderWith(<JournalReviewPage />);

    expect(
      await screen.findByText(/needs the Journal Entry approve permission/),
    ).toBeInTheDocument();
  });
});

describe("journal review columns", () => {
  it("links each entry to its record and names its source", () => {
    const row = {
      id: "je_7",
      entryNumber: "JE-7",
      status: "Pending",
      referenceType: "CreditMemoPosted",
      referenceNumber: "CM-12",
    } as JournalReviewRow;
    const columns = getJournalReviewColumns(t, labels);
    const cell = (key: string) => {
      const column = columns.find(
        (candidate) => "accessorKey" in candidate && candidate.accessorKey === key,
      );
      const render = column?.cell as (context: {
        row: { original: JournalReviewRow };
      }) => ReactNode;
      return render({ row: { original: row } });
    };

    renderWith(<>{cell("entryNumber") as ReactElement}</>);
    expect(screen.getByRole("link", { name: "JE-7" })).toHaveAttribute(
      "href",
      recordPath("journal_entry", "je_7"),
    );
    expect(cell("referenceType")).toBe("Credit memo posted");
  });
});
