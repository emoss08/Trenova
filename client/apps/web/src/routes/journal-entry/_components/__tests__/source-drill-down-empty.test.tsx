import { cleanup, screen } from "@testing-library/react";
import { Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageLayoutStub, renderAccountingPage } from "@/test/accounting-page-mocks";
import { SourceDrillDownPage } from "../source-drill-down-page";

const mocks = vi.hoisted(() => ({
  fetchJournalEntriesBySource: vi.fn(),
  fetchJournalSourceByObject: vi.fn(),
}));

vi.mock("@/lib/graphql/journal-entry", () => ({
  fetchJournalEntriesBySource: mocks.fetchJournalEntriesBySource,
  fetchJournalSourceByObject: mocks.fetchJournalSourceByObject,
  fetchJournalEntry: vi.fn(),
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({ PageLayout: PageLayoutStub }));
vi.mock("@/components/accounting/source-drill-down-link", () => ({
  SourceDrillDownLink: () => null,
}));

beforeEach(() => {
  mocks.fetchJournalEntriesBySource.mockResolvedValue([]);
  mocks.fetchJournalSourceByObject.mockResolvedValue(null);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("journal source drill-down empty state", () => {
  it("draws the postings it will show and names the source that has none", async () => {
    renderAccountingPage(
      <Routes>
        <Route path="/journal/source/:type/:sourceId" element={<SourceDrillDownPage />} />
      </Routes>,
      ["/journal/source/invoice/inv_1"],
    );

    expect(await screen.findByText("Nothing posted")).toBeInTheDocument();
    expect(screen.getByText(/for this invoice yet/)).toBeInTheDocument();
  });
});
