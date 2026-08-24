import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { RateImportBatch } from "@trenova/shared/types/rate";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ImportRateSheetDialog } from "../import-rate-sheet-dialog";

const template = vi.fn();
const upload = vi.fn();
const commit = vi.fn();
const discard = vi.fn();
const listFailedRows = vi.fn().mockResolvedValue([]);
const downloadCsv = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    get rateImportService() {
      return { template, upload, commit, discard, listFailedRows };
    },
  },
}));

vi.mock("@/lib/data-table-export", () => ({
  downloadCsv: (...args: unknown[]) => downloadCsv(...args),
}));

// The autocomplete talks to the network; the dialog's contract only needs a
// field that holds the chosen agreement id.
vi.mock("@/components/autocomplete-fields", () => ({
  RateAgreementAutocompleteField: ({ control, name }: { control: Control; name: string }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label="Agreement"
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: () => null,
}));

function renderDialog() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ImportRateSheetDialog open onOpenChange={() => {}} />
    </QueryClientProvider>,
  );
}

async function chooseAgreementAndUpload() {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText("Agreement"), "ragr_1");

  // The dialog renders in a portal, so the hidden input is found on the
  // document rather than the render container.
  const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
  const file = new File(["Origin,Destination\nGA,FL"], "rates.csv", { type: "text/csv" });
  await user.upload(fileInput, file);
}

const stagedBatch = {
  rateAgreementId: "ragr_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  id: "rimp_1",
  fileName: "rates.csv",
  sourceFormat: "CSV",
  status: "Parsed",
  effectiveFrom: 1_700_000_000,
  rowCount: 2,
  errorCount: 0,
  error: "",
  unmappedHeaders: [],
  summary: { added: 2, changed: 0, removed: 0, duplicate: 0, unchanged: 0 },
  changes: [
    { kind: "Added", laneKey: "ST:GA>ST:FL", label: "GA to FL", fields: [] },
    { kind: "Added", laneKey: "ST:GA>ST:TX", label: "GA to TX", fields: [] },
  ],
} as unknown as RateImportBatch;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ImportRateSheetDialog", () => {
  it("hands the user the template the importer recognises", async () => {
    template.mockResolvedValue({
      fileName: "rate-sheet-template.csv",
      content: "Origin,Destination,Rate\n",
    });
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /download template/i }));

    await waitFor(() => {
      expect(downloadCsv).toHaveBeenCalledWith(
        "Origin,Destination,Rate\n",
        "rate-sheet-template.csv",
      );
    });
  });

  it("shows every template problem the server reported, not just the first", async () => {
    upload.mockRejectedValue(
      new ApiRequestError(422, {
        type: "validation-error",
        title: "Validation Error",
        status: 422,
        errors: [
          { field: "file", message: "this sheet has no rate column" },
          { field: "file", message: "this sheet does not say where lanes end" },
        ],
      }),
    );
    renderDialog();

    await chooseAgreementAndUpload();

    expect(await screen.findByText("this sheet has no rate column")).toBeInTheDocument();
    expect(screen.getByText("this sheet does not say where lanes end")).toBeInTheDocument();
  });

  it("stages a dry run and applies it only on the second, deliberate click", async () => {
    upload.mockResolvedValue(stagedBatch);
    commit.mockResolvedValue({ ...stagedBatch, status: "Committed" });
    const user = userEvent.setup();
    renderDialog();

    await chooseAgreementAndUpload();

    expect(
      await screen.findByText(/committing this would leave the agreement with 2 new/i),
    ).toBeInTheDocument();
    expect(screen.getByText("GA to FL")).toBeInTheDocument();
    expect(commit).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /apply these rates/i }));

    await waitFor(() => expect(commit).toHaveBeenCalledWith("rimp_1"));
  });
});
