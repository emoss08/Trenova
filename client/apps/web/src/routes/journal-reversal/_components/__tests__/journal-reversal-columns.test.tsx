import { cleanup, render, screen } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { JournalReversalRow } from "@/lib/graphql/journal-reversal-table";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { getColumns } from "../journal-reversal-columns";

afterEach(cleanup);

type Meta = { apiField?: string };

const row: JournalReversalRow = {
  id: "jrev_01",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  originalJournalEntryId: "je_01JD8Q3X",
  originalJournalEntry: {
    id: "je_01JD8Q3X",
    fiscalPeriodId: "fp_1",
    entryNumber: "JE-2026-000418",
    entryType: "Standard",
    status: "Posted",
    accountingDate: 1_792_108_800,
  },
  reversalJournalEntryId: null,
  reversalJournalEntry: null,
  postedBatchId: null,
  status: "PendingApproval",
  requestedAccountingDate: 1_792_195_200,
  resolvedFiscalYearId: "fy_1",
  resolvedFiscalPeriodId: "fp_1",
  reasonCode: "DUPLICATE",
  reasonText: "Billed twice.",
  requestedById: "usr_1",
  approvedById: null,
  approvedAt: null,
  rejectedById: null,
  rejectedAt: null,
  rejectionReason: null,
  cancelledById: null,
  cancelledAt: null,
  cancelReason: null,
  postedById: null,
  postedAt: null,
  version: 1,
  createdAt: 1_792_150_000,
  updatedAt: 1_792_150_000,
};

function originalEntryColumn(): ColumnDef<JournalReversalRow> {
  const column = getColumns(translate).find(
    (candidate) =>
      (candidate.meta as Meta | undefined)?.apiField === "originalJournalEntry.entryNumber",
  );
  if (!column) {
    throw new Error("no original journal entry column");
  }
  return column;
}

describe("journal reversal table original entry column", () => {
  it("names the original entry by its entry number and links to it, never showing its id", () => {
    const cell = originalEntryColumn().cell as (context: {
      row: { original: JournalReversalRow };
    }) => ReactNode;
    render(<MemoryRouter>{cell({ row: { original: row } })}</MemoryRouter>);

    const link = screen.getByRole("link");
    expect(link.textContent).toBe("JE-2026-000418");
    expect(link.getAttribute("href")).toBe("/accounting/journal-entries/je_01JD8Q3X");
    expect(screen.queryByText("je_01JD8Q3X")).toBeNull();
  });

  it("filters and sorts by the entry number, matching part of it", () => {
    expect(originalEntryColumn().meta).toMatchObject({
      apiField: "originalJournalEntry.entryNumber",
      filterable: true,
      sortable: true,
      filterType: "text",
      defaultFilterOperator: "contains",
    });
  });
});
