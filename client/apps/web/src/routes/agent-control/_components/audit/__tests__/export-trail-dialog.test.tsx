import type { AIAuditDownload, AIAuditExport } from "@/lib/graphql/ai-audit";
import type { RequestAiAuditExportInput } from "@trenova/graphql/generated/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DEFAULT_AUDIT_SCOPE, type AuditTableState, type AuditTrailScope } from "../audit-model";

const requestAIAuditExport = vi.fn<(input: RequestAiAuditExportInput) => Promise<AIAuditExport>>();
const fetchAIAuditExportDownload = vi.fn<(id: string) => Promise<AIAuditDownload>>();
const downloadFromUrl = vi.fn<(url: string, fileName?: string) => void>();

vi.mock("@/lib/graphql/ai-audit", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-audit")>();
  return {
    ...actual,
    requestAIAuditExport: (input: RequestAiAuditExportInput) => requestAIAuditExport(input),
    fetchAIAuditExportDownload: (id: string) => fetchAIAuditExportDownload(id),
  };
});

vi.mock("@trenova/shared/lib/utils", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/utils")>();
  return {
    ...actual,
    downloadFromUrl: (url: string, fileName?: string) => downloadFromUrl(url, fileName),
  };
});

const { ExportTrailDialog } = await import("../export-trail-dialog");

/** An export as requestAIAuditExport returns it, per the AIAuditExport type. */
function exportRecord(overrides: Partial<AIAuditExport> = {}): AIAuditExport {
  return {
    id: "aiax_1",
    requestedByUserId: "usr_me",
    requestedBy: {
      id: "usr_me",
      name: "Ada Admin",
      username: "ada",
      profilePicUrl: null,
      thumbnailUrl: null,
    },
    format: "CSV",
    filters: null,
    rangeFrom: 1_758_000_000,
    rangeTo: 1_758_600_000,
    snapshotSeq: 1_204,
    status: "Succeeded",
    rowCount: 42,
    byteSize: 2_048,
    sha256: "f".repeat(64),
    artifactExpiresAt: 1_759_200_000,
    chainKeyId: "k2026",
    chainFirstSeq: 10,
    chainLastSeq: 51,
    chainComplete: false,
    errorMessage: null,
    startedAt: 1_758_600_001,
    completedAt: 1_758_600_002,
    downloadable: true,
    createdAt: 1_758_600_000,
    updatedAt: 1_758_600_002,
    ...overrides,
  } as AIAuditExport;
}

const scope: AuditTrailScope = {
  ...DEFAULT_AUDIT_SCOPE,
  agent: { id: "agdef_1", name: "Billing desk" },
  personId: "usr_1",
};

const table: AuditTableState = {
  query: "",
  fieldFilters: [{ field: "kind", operator: "eq", value: "ToolCall" }],
  filterGroups: [],
  sort: [],
};

const onOpenExports = vi.fn();

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ExportTrailDialog
        open
        onOpenChange={() => {}}
        scope={scope}
        table={table}
        onOpenExports={onOpenExports}
      />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  requestAIAuditExport.mockReset();
  fetchAIAuditExportDownload.mockReset();
  downloadFromUrl.mockReset();
  onOpenExports.mockReset();
});

afterEach(() => {
  cleanup();
});

describe("ExportTrailDialog", () => {
  // A small export comes back already written; the person gets the file at
  // once, through the one-minute link only the requester is given.
  it("downloads a file written while the person waits", async () => {
    const user = userEvent.setup();
    requestAIAuditExport.mockResolvedValue(exportRecord());
    fetchAIAuditExportDownload.mockResolvedValue({
      url: "https://files.example/aiax_1.csv?sig=1",
      fileName: "ai-audit-aiax_1.csv",
      expiresAt: 1_758_600_060,
      sha256: "f".repeat(64),
    });
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Export" }));

    expect(await screen.findByText("The file is ready")).toBeInTheDocument();
    expect(fetchAIAuditExportDownload).toHaveBeenCalledWith("aiax_1");
    expect(downloadFromUrl).toHaveBeenCalledWith(
      "https://files.example/aiax_1.csv?sig=1",
      "ai-audit-aiax_1.csv",
    );
    expect(screen.getByText(/Filters left rows out/)).toBeInTheDocument();

    const input = requestAIAuditExport.mock.calls[0][0];
    expect(input.format).toBe("CSV");
    expect(input.to).toBeGreaterThan(input.from);
    expect(input.fieldFilters).toEqual([
      { field: "agentDefinitionId", operator: "eq", value: "agdef_1" },
      { field: "personId", operator: "eq", value: "usr_1" },
      { field: "purpose", operator: "eq", value: "Live" },
      { field: "kind", operator: "eq", value: "ToolCall" },
    ]);
  });

  // A large export is written in the background: nothing is downloaded now,
  // and the person is told where it will be.
  it("says a large export is running and offers the exports list", async () => {
    const user = userEvent.setup();
    requestAIAuditExport.mockResolvedValue(
      exportRecord({
        status: "Running",
        rowCount: 0,
        byteSize: 0,
        sha256: null,
        downloadable: false,
      }),
    );
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Export" }));

    expect(await screen.findByText("Running — you'll be notified")).toBeInTheDocument();
    expect(fetchAIAuditExportDownload).not.toHaveBeenCalled();
    expect(downloadFromUrl).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Open exports" }));
    expect(onOpenExports).toHaveBeenCalledTimes(1);
  });

  // The server refuses an export over its row cap with a validation error
  // naming the count; the dialog shows that message and stays open.
  it("shows why the server refused the export", async () => {
    const user = userEvent.setup();
    const message =
      "This export would hold 1200000 rows; one export can hold at most 1000000. Narrow the range or the filters.";
    requestAIAuditExport.mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        message,
        graphQLErrors: [{ message, extensions: { code: "BAD_USER_INPUT" } }],
      }),
    );
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Export" }));

    expect(await screen.findByText(message)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Export" })).toBeEnabled();
    expect(downloadFromUrl).not.toHaveBeenCalled();
  });

  it("says when a written file could not be downloaded, and lets it be tried again", async () => {
    const user = userEvent.setup();
    requestAIAuditExport.mockResolvedValue(exportRecord({ chainComplete: true }));
    fetchAIAuditExportDownload.mockRejectedValueOnce(
      new Error("Only the person who asked for an export may download it"),
    );
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Export" }));

    expect(
      await screen.findByText("Only the person who asked for an export may download it"),
    ).toBeInTheDocument();
    expect(screen.getByText(/can be checked end to end/)).toBeInTheDocument();

    fetchAIAuditExportDownload.mockResolvedValueOnce({
      url: "https://files.example/again",
      fileName: "again.csv",
      expiresAt: 1,
      sha256: "f".repeat(64),
    });
    await user.click(screen.getByRole("button", { name: "Try the download again" }));
    await waitFor(() =>
      expect(downloadFromUrl).toHaveBeenCalledWith("https://files.example/again", "again.csv"),
    );
  });

  it("sends only the range and format when current filters are turned off", async () => {
    const user = userEvent.setup();
    requestAIAuditExport.mockResolvedValue(exportRecord({ status: "Running" }));
    renderDialog();

    await user.click(screen.getByRole("checkbox", { name: "Use current filters" }));
    await user.click(screen.getByRole("radio", { name: /JSON/ }));
    await user.click(screen.getByRole("button", { name: "Export" }));

    await waitFor(() => expect(requestAIAuditExport).toHaveBeenCalledTimes(1));
    const input = requestAIAuditExport.mock.calls[0][0];
    expect(Object.keys(input).sort()).toEqual(["format", "from", "to"]);
    expect(input.format).toBe("JSON");
  });
});
