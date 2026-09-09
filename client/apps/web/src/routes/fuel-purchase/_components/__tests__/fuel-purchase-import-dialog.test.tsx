import type { FuelPurchaseImportBatch } from "@/lib/graphql/fuel-purchase-import";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FuelPurchaseImportDialog } from "../import/fuel-purchase-import-dialog";

const {
  createFuelPurchaseImport,
  stageFuelPurchaseImport,
  commitFuelPurchaseImport,
  discardFuelPurchaseImport,
  fetchFuelPurchaseImportRows,
  fetchFuelPurchaseImportTemplate,
  useDocumentUpload,
  downloadCsv,
} = vi.hoisted(() => ({
  createFuelPurchaseImport: vi.fn(),
  stageFuelPurchaseImport: vi.fn(),
  commitFuelPurchaseImport: vi.fn(),
  discardFuelPurchaseImport: vi.fn(),
  fetchFuelPurchaseImportRows: vi.fn(),
  fetchFuelPurchaseImportTemplate: vi.fn(),
  useDocumentUpload: vi.fn(),
  downloadCsv: vi.fn(),
}));

vi.mock("@/lib/graphql/fuel-purchase-import", () => ({
  createFuelPurchaseImport,
  stageFuelPurchaseImport,
  commitFuelPurchaseImport,
  discardFuelPurchaseImport,
  fetchFuelPurchaseImportRows,
  fetchFuelPurchaseImportTemplate,
  FUEL_PURCHASE_IMPORT_KEY: "fuel-purchase-import",
  FUEL_PURCHASE_IMPORT_ROWS_KEY: "fuel-purchase-import-rows",
}));

vi.mock("@/lib/graphql/fuel-purchase", () => ({
  FUEL_PURCHASE_LIST_KEY: "fuel-purchase-list",
}));

vi.mock("@/hooks/use-document-upload", () => ({
  useDocumentUpload: (options: unknown) => useDocumentUpload(options),
}));

vi.mock("@/lib/data-table-export", () => ({
  downloadCsv: (...args: unknown[]) => downloadCsv(...args),
}));

vi.mock("@/components/documents/document-upload-zone", () => ({
  DocumentUploadZone: ({
    onFilesSelected,
    disabled,
  }: {
    onFilesSelected: (files: File[]) => void;
    disabled?: boolean;
  }) => (
    <button
      type="button"
      disabled={disabled}
      onClick={() =>
        onFilesSelected([new File(["a,b\n1,2"], "statement.csv", { type: "text/csv" })])
      }
    >
      Choose statement
    </button>
  ),
}));

vi.mock("@/components/autocomplete-fields", () => ({
  FuelCardAutocompleteField: ({
    control,
    name,
    label,
  }: {
    control: Control;
    name: string;
    label: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({
    control,
    name,
    label,
    options,
  }: {
    control: Control;
    name: string;
    label: string;
    options: { value: string; label: string }[];
  }) => {
    const { field } = useController({ control, name });
    return (
      <select
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value || null)}
      >
        <option value="">—</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    );
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

function batch(overrides: Partial<FuelPurchaseImportBatch> = {}): FuelPurchaseImportBatch {
  return {
    id: "fpib_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    provider: "Comdata",
    documentId: null,
    fileName: null,
    sourceFormat: null,
    status: "Pending",
    defaultFuelType: "Diesel",
    defaultFuelCardId: null,
    defaultCurrency: "USD",
    mapping: null,
    unmappedHeaders: [],
    summary: null,
    rowCount: 0,
    errorCount: 0,
    committedCount: 0,
    error: null,
    uploadedById: "usr_1",
    stagedAt: null,
    committedAt: null,
    committedById: null,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    document: null,
    defaultFuelCard: null,
    ...overrides,
  };
}

const parsed = batch({
  status: "Parsed",
  documentId: "doc_1",
  fileName: "statement.csv",
  sourceFormat: "CSV",
  rowCount: 3,
  version: 2,
  summary: {
    rowCount: 3,
    newCount: 2,
    duplicateInFileCount: 0,
    alreadyImportedCount: 1,
    errorCount: 0,
    totalGallons: "250.000",
    totalAmount: "900.00",
    byFuelType: null,
    byJurisdiction: null,
    earliestPurchasedAt: 1_700_000_000,
    latestPurchasedAt: 1_700_100_000,
  },
});

function rowsPage(
  rows: Array<{ id: string; rowNumber: number; status: string; error?: string | null }>,
) {
  return {
    edges: rows.map((row) => ({
      cursor: row.id,
      node: {
        id: row.id,
        importBatchId: "fpib_1",
        rowNumber: row.rowNumber,
        cells: [],
        parsed: null,
        transactionReference: `REF-${row.rowNumber}`,
        status: row.status,
        error: row.error ?? null,
        resolvedTractorId: null,
        resolvedFuelCardId: null,
        resolvedJurisdictionId: null,
        resolutionNotes: [],
        fuelPurchaseId: null,
        createdAt: 1,
        resolvedTractor: null,
      },
    })),
    totalCount: rows.length,
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function armUploadHook() {
  useDocumentUpload.mockImplementation((options: { onSuccess: (doc: unknown) => void }) => ({
    uploads: [],
    uploadFiles: (files: File[]) => {
      options.onSuccess({
        id: "doc_1",
        originalName: files[0]?.name ?? "statement.csv",
        fileSize: 12,
      });
    },
    cancelUpload: vi.fn(),
    retryUpload: vi.fn(),
    removeUpload: vi.fn(),
    clearCompleted: vi.fn(),
    clearAll: vi.fn(),
    isUploading: false,
  }));
}

function renderDialog() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <FuelPurchaseImportDialog open onOpenChange={onOpenChange} />
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

async function stageStatement(staged: FuelPurchaseImportBatch) {
  createFuelPurchaseImport.mockResolvedValue(batch());
  stageFuelPurchaseImport.mockResolvedValue(staged);
  armUploadHook();
  fireEvent.click(screen.getByRole("button", { name: "Choose statement" }));
  await waitFor(() => expect(stageFuelPurchaseImport).toHaveBeenCalled());
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("FuelPurchaseImportDialog", () => {
  it("creates the batch, uploads against it, then stages with the document it produced", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(rowsPage([]));
    renderDialog();

    await stageStatement(parsed);

    expect(createFuelPurchaseImport).toHaveBeenCalledWith({
      provider: "Comdata",
      defaultFuelCardId: null,
      defaultFuelType: "Diesel",
      defaultCurrency: "USD",
    });
    expect(useDocumentUpload).toHaveBeenCalledWith(
      expect.objectContaining({ resourceId: "fpib_1", resourceType: "fuel_purchase_import" }),
    );
    expect(stageFuelPurchaseImport).toHaveBeenCalledWith({ id: "fpib_1", documentId: "doc_1" });
    expect(
      await screen.findByText(
        "Importing would record 2 purchases for 250.000 gallons and $900.00.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText("1 row matches a purchase already on file and will be skipped."),
    ).toBeInTheDocument();
  });

  it("enables the commit only for a parsed batch with new rows, and commits with its version", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(rowsPage([]));
    commitFuelPurchaseImport.mockResolvedValue({
      ...parsed,
      status: "Committed",
      committedCount: 2,
      version: 3,
    });
    renderDialog();

    await stageStatement(parsed);

    const commitButton = await screen.findByRole("button", { name: "Import 2 purchases" });
    expect(commitButton).toBeEnabled();
    fireEvent.click(commitButton);

    await waitFor(() => expect(commitFuelPurchaseImport).toHaveBeenCalledWith("fpib_1", 2));
    expect(await screen.findByText("Recorded 2 purchases.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("keeps the commit disabled when nothing in the statement is new", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(rowsPage([]));
    renderDialog();

    await stageStatement({
      ...parsed,
      summary: { ...parsed.summary!, newCount: 0, alreadyImportedCount: 3 },
    });

    expect(await screen.findByRole("button", { name: "Import purchases" })).toBeDisabled();
    expect(commitFuelPurchaseImport).not.toHaveBeenCalled();
  });

  it("asks before throwing away a parsed batch, then discards it and closes", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(rowsPage([]));
    discardFuelPurchaseImport.mockResolvedValue({ ...parsed, status: "Discarded", version: 3 });
    const { onOpenChange } = renderDialog();

    await stageStatement(parsed);
    await screen.findByRole("button", { name: "Import 2 purchases" });

    const dialogs = screen.getAllByRole("dialog");
    const closeButton = within(dialogs[0]).queryByRole("button", { name: /close/i });
    if (closeButton) {
      fireEvent.click(closeButton);
    } else {
      fireEvent.keyDown(dialogs[0], { key: "Escape" });
    }

    expect(await screen.findByText("Discard this import?")).toBeInTheDocument();
    expect(
      screen.getByText(
        "The 3 rows read from statement.csv will be thrown away. Nothing has been recorded, and the statement can be uploaded again.",
      ),
    ).toBeInTheDocument();
    expect(discardFuelPurchaseImport).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Discard import" }));

    await waitFor(() => expect(discardFuelPurchaseImport).toHaveBeenCalledWith("fpib_1", 2));
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("discarding from the footer also asks first and then shows the discarded state", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(rowsPage([]));
    discardFuelPurchaseImport.mockResolvedValue({ ...parsed, status: "Discarded", version: 3 });
    const { onOpenChange } = renderDialog();

    await stageStatement(parsed);
    fireEvent.click(await screen.findByRole("button", { name: "Discard" }));

    expect(await screen.findByText("Discard this import?")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Discard import" }));

    await waitFor(() => expect(discardFuelPurchaseImport).toHaveBeenCalledWith("fpib_1", 2));
    expect(
      await screen.findByText("This import was discarded; nothing was recorded."),
    ).toBeInTheDocument();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });

  it("hands the user the provider's template", async () => {
    fetchFuelPurchaseImportTemplate.mockResolvedValue({
      fileName: "comdata-statement-template.csv",
      content: "Card,Date,Amount\n",
    });
    renderDialog();

    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "EFS" } });
    fireEvent.click(screen.getByRole("button", { name: /download template/i }));

    await waitFor(() => expect(fetchFuelPurchaseImportTemplate).toHaveBeenCalledWith("EFS"));
    await waitFor(() =>
      expect(downloadCsv).toHaveBeenCalledWith(
        "Card,Date,Amount\n",
        "comdata-statement-template.csv",
      ),
    );
  });

  it("lists every error when the statement could not be read", async () => {
    fetchFuelPurchaseImportRows.mockResolvedValue(
      rowsPage([
        { id: "r1", rowNumber: 2, status: "Error", error: "date could not be parsed" },
        { id: "r2", rowNumber: 5, status: "Error", error: "unknown tractor code 999" },
      ]),
    );
    renderDialog();

    await stageStatement({
      ...parsed,
      status: "Failed",
      error: "The header row names no amount column",
      rowCount: 5,
      errorCount: 2,
      summary: null,
    });

    expect(await screen.findByText("The statement could not be read.")).toBeInTheDocument();
    expect(screen.getByText(/The header row names no amount column/)).toBeInTheDocument();
    expect(await screen.findByText("date could not be parsed")).toBeInTheDocument();
    expect(screen.getByText("unknown tractor code 999")).toBeInTheDocument();
    expect(fetchFuelPurchaseImportRows).toHaveBeenCalledWith(
      "fpib_1",
      expect.objectContaining({ statuses: ["Error"] }),
      expect.anything(),
    );
    expect(screen.getByRole("button", { name: "Try another file" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /import .*purchases/i })).not.toBeInTheDocument();
  });
});
